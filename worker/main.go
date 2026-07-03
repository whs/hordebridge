package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/go-faster/errors"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/whs/hordebridge/aihorde"
	"github.com/whs/hordebridge/worker/inference"
	"github.com/whs/hordebridge/worker/inference/openresponses"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/semconv/v1.41.0/messagingconv"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sync/errgroup"
)

var errContentClassifier = errors.New("blocked by content classifier")

type Worker struct {
	config           Config
	logger           *slog.Logger
	aihorde          *aihorde.Client
	openai           openai.Client
	openaiClassifier openai.Client
	completion       inference.TextInference
	submitWg         sync.WaitGroup
	tracer           trace.Tracer

	metricClientConsumedMessages messagingconv.ClientConsumedMessages
	metricJobProcessingDuration  messagingconv.ProcessDuration
	metricInputLength            metric.Int64Counter
	metricOutputLength           metric.Int64Counter
	metricClassifierResult       metric.Int64Counter
	metricClassifierDuration     metric.Float64Histogram
	metricKudos                  metric.Float64Counter
}

func NewWorker(config Config) (*Worker, error) {
	httpTransport := otelhttp.NewTransport(nil)
	httpClient := &http.Client{
		Transport: httpTransport,
	}

	aihordeClient, err := aihorde.NewClient(config.HordeServer, aihorde.WithClient(httpClient))
	if err != nil {
		return nil, err
	}

	openaiClient := openai.NewClient(option.WithAPIKey(config.OpenaiAPIKey), option.WithBaseURL(config.OpenaiServer), option.WithHTTPClient(httpClient))

	if len(config.AdditionalParams) > 0 {
		if !json.Valid([]byte(config.AdditionalParams)) {
			return nil, fmt.Errorf("additional params is not json")
		}
	}
	if len(config.ResponsesAdditionalParams) > 0 {
		if !json.Valid([]byte(config.ResponsesAdditionalParams)) {
			return nil, fmt.Errorf("responses additional params is not json")
		}
	}

	var openaiClassifier openai.Client
	if config.Classifier.UseClassifier() {
		if config.Classifier.APIKey == "" {
			config.Classifier.APIKey = config.OpenaiAPIKey
		}
		if config.Classifier.Server == "" {
			config.Classifier.Server = config.OpenaiServer
		}
		if config.Classifier.Model == "" {
			config.Classifier.Server = config.OpenaiModel
		}
		openaiClassifier = openai.NewClient(option.WithAPIKey(config.Classifier.APIKey), option.WithBaseURL(config.Classifier.Server), option.WithHTTPClient(httpClient))

		if config.Classifier.BlockNSFW && config.NSFW {
			return nil, fmt.Errorf("--nsfw is allowed, but --classifier-block-nsfw is on")
		}
	}

	var completion inference.TextInference
	completion, err = inference.NewOpenAICompletion(openaiClient, inference.OpenAICompletionConfig{
		Model:            config.OpenaiModel,
		AdditionalParams: []byte(config.AdditionalParams),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create openai client: %w", err)
	}

	if config.ResponsesAPI {
		slog.Info("Creating worker with responses API parsing")
		completion, err = openresponses.New(openaiClient, openresponses.ResponsesConfig{
			Model:            config.OpenaiModel,
			Fallback:         completion,
			AdditionalParams: []byte(config.ResponsesAdditionalParams),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create openresponses client: %w", err)
		}
	}

	w := &Worker{
		config:           config,
		logger:           slog.Default().With("module", "worker"),
		aihorde:          aihordeClient,
		openai:           openaiClient,
		openaiClassifier: openaiClassifier,
		completion:       completion,
		tracer:           otel.Tracer(otelPackageName),
	}
	err = w.initOtel()
	if err != nil {
		return nil, err
	}

	return w, nil
}

// Start the main worker loop
// The worker loop is defined in https://github.com/Haidra-Org/haidra-assets/blob/main/docs/workers.md
func (w *Worker) Start(ctx context.Context, abortCtx context.Context) {
	waitCount := 0
	errorCount := 0

	sleep := func(dur time.Duration) {
		select {
		case <-ctx.Done():
		case <-time.After(dur):
		}
	}

	defer w.submitWg.Wait()

	onError := func(err error) bool {
		errorCount += 1

		if errorCount >= w.config.QuitAfterErrors {
			w.logger.ErrorContext(ctx, "Too much error, exiting")
			return true
		}

		sleep(1 * time.Minute)
		return false
	}

	for {
		if ctx.Err() != nil {
			if !errors.Is(ctx.Err(), context.Canceled) {
				w.logger.DebugContext(ctx, "Context error", "err", ctx.Err())
			}
			break
		}
		job, err := w.GetJob(ctx)
		if err != nil {
			w.logger.ErrorContext(ctx, "Failed to get job", "err", err)
			if onError(err) {
				return
			}
			continue
		}

		for _, message := range job.Messages {
			w.logger.WarnContext(ctx, "Job message", "message", message.Message, "origin", message.Origin, "id", message.ID)
		}

		if job.ID.IsNull() || !job.ID.IsSet() {
			var waitTime time.Duration
			if waitCount < 10 {
				waitTime = 1 * time.Second
			} else if waitCount < 25 {
				waitTime = 2 * time.Second
			} else {
				waitTime = 3 * time.Second
			}
			w.logger.DebugContext(ctx, "No job available", "wait", waitTime)
			sleep(waitTime)
			waitCount += 1
			continue
		}

		// Got a job!
		waitCount = 0
		jobCtx, jobSpan := w.tracer.Start(ctx, "consume TextJob", trace.WithSpanKind(trace.SpanKindConsumer), trace.WithAttributes(
			attribute.String("messaging.message.id", job.ID.Value),
		))

		err = w.ProcessJob(jobCtx, job)
		if err != nil {
			jobSpan.RecordError(err)
			jobSpan.SetStatus(codes.Error, err.Error())
			_, jobErrSpan := w.tracer.Start(jobCtx, "Send error")
			jobAbortCtx := trace.ContextWithSpan(abortCtx, jobErrSpan)
			w.logger.ErrorContext(jobAbortCtx, "Failed to process job. Sending error", "err", err)

			reportable, ok := errors.Into[ReportableError](err)
			// XXX: Use abortCtx to ensure that if ctx is canceled, this job should be able to send the report
			if ok {
				jobErrSpan.SetAttributes(keyKoboldState.String(string(reportable.Kind)))
				sendErrErr := w.SubmitJob(jobAbortCtx, &aihorde.SubmitInputKobold{
					ID:         job.ID.Value,
					Generation: reportable.PublicError,
					State:      aihorde.NewOptSubmitInputKoboldState(reportable.Kind),
				})
				if sendErrErr != nil {
					w.logger.ErrorContext(jobAbortCtx, "Failed to send job error. Exiting", "err", sendErrErr, "originalErr", err)
					jobErrSpan.End()
					jobSpan.End()
					return
				}
			} else {
				jobErrSpan.SetAttributes(keyKoboldState.String(string(aihorde.SubmitInputKoboldStateFaulted)))
				sendErrErr := w.SubmitJob(jobAbortCtx, &aihorde.SubmitInputKobold{
					ID:         job.ID.Value,
					Generation: "[Worker error]",
					State:      aihorde.NewOptSubmitInputKoboldState(aihorde.SubmitInputKoboldStateFaulted),
				})
				if sendErrErr != nil {
					w.logger.ErrorContext(jobAbortCtx, "Failed to send job error. Exiting", "err", sendErrErr, "originalErr", err)
					jobErrSpan.End()
					jobSpan.End()
					return
				}
			}

			if onError(err) {
				jobErrSpan.End()
				jobSpan.End()
				return
			}
			jobErrSpan.End()
			jobSpan.End()
			continue
		} else {
			w.metricClientConsumedMessages.Add(jobCtx, 1, "TextJob", messagingSystemHorde,
				w.metricClientConsumedMessages.AttrConsumerGroupName(w.config.WorkerName),
				w.metricClientConsumedMessages.AttrDestinationName(w.config.HordeModel),
			)
			jobSpan.SetStatus(codes.Ok, "")
		}

		jobSpan.End()
		sleep(100 * time.Millisecond)
		errorCount = 0
	}
}

func (w *Worker) GetJob(ctx context.Context) (*aihorde.GenerationPayloadKobold, error) {
	resp, err := w.aihorde.PostTextJobPop(ctx, &aihorde.PopInputKobold{
		Name:                aihorde.NewOptString(w.config.WorkerName),
		PriorityUsernames:   w.config.PriorityUsernames,
		Nsfw:                aihorde.NewOptBool(w.config.NSFW),
		Models:              []string{w.config.HordeModel},
		BridgeAgent:         aihorde.NewOptString(w.config.BridgeAgent),
		Threads:             aihorde.NewOptInt(1),
		RequireUpfrontKudos: aihorde.NewOptBool(w.config.RequireUpfrontKudos),
		Amount:              aihorde.NewOptInt(1),
		ExtraSlowWorker:     aihorde.NewOptBool(w.config.ExtraSlowWorker),
		MaxLength:           aihorde.NewOptInt(w.config.MaxLength),
		MaxContextLength:    aihorde.NewOptInt(w.config.MaxContextLength),
	}, aihorde.PostTextJobPopParams{
		Apikey: w.config.HordeAPIKey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get job: %w", err)
	}

	switch job := resp.(type) {
	case *aihorde.GenerationPayloadKobold:
		return job, nil
	default:
		return nil, fmt.Errorf("unknown response type %+v", resp)
	}
}

func (w *Worker) ProcessJob(parentCtx context.Context, job *aihorde.GenerationPayloadKobold) error {
	logger := w.logger.With("jobId", job.ID.Value)
	span := trace.SpanFromContext(parentCtx)

	ctx, cancel := context.WithTimeout(parentCtx, time.Duration(job.TTL.Or(60*60))*time.Second)
	defer cancel()

	payload, ok := job.Payload.Get()
	if !ok {
		return fmt.Errorf("no job payload")
	}

	w.metricInputLength.Add(ctx, int64(len(payload.Prompt.Value)))
	span.AddEvent("gen_ai.client.inference.operation.details", trace.WithAttributes(inputToEventAttr(&payload)...))

	if maxLength, ok := payload.MaxLength.Get(); ok && maxLength > w.config.MaxLength {
		logger.InfoContext(ctx, "Truncating max_length", "original", maxLength, "new", w.config.MaxLength)
		payload.MaxLength.Value = w.config.MaxLength
	}
	// TODO: Truncate input to context window (max_context_length)

	var generation string
	var classifiedResult ClassifierResult

	errGroup, errCtx := errgroup.WithContext(ctx)
	generationCtx, generationCtxCancel := context.WithCancelCause(errCtx)
	defer generationCtxCancel(context.Canceled)

	errGroup.Go(func() error {
		var err error

		logger.InfoContext(errCtx, "Running text generation", "inputLen", len(payload.Prompt.Value))
		generateTextCtx, generateTextSpan := w.tracer.Start(generationCtx, "GenerateText", trace.WithAttributes(keyInputLength.Int(len(payload.Prompt.Value))))
		defer generateTextSpan.End()

		start := time.Now()
		generation, err = w.completion.GenerateText(generateTextCtx, job)
		finish := time.Since(start)
		w.metricClassifierDuration.Record(generateTextCtx, finish.Seconds())
		w.metricOutputLength.Add(generationCtx, int64(len(generation)))

		if errors.Is(err, errContentClassifier) || errors.Is(err, context.Canceled) {
			generateTextSpan.SetStatus(codes.Error, err.Error())
			return nil
		}

		generateTextSpan.RecordError(err)
		if err != nil {
			generateTextSpan.SetStatus(codes.Error, err.Error())
		} else {
			generateTextSpan.SetStatus(codes.Ok, "")
		}
		return err
	})

	if w.config.Classifier.UseClassifier() {
		errGroup.Go(func() error {
			logger.InfoContext(errCtx, "Running content classifier")
			classifierTimeoutCtx, classifierTimeoutCancel := context.WithTimeout(errCtx, w.config.Classifier.Timeout)
			defer classifierTimeoutCancel()

			classifierCtx, classifierSpan := w.tracer.Start(classifierTimeoutCtx, "ClassifyContent", trace.WithAttributes(keyInputLength.Int(len(payload.Prompt.Value))))
			defer classifierSpan.End()

			start := time.Now()
			classifiedResult = w.ClassifyContent(classifierCtx, payload.Prompt.Value, "")
			finish := time.Since(start)

			logger.InfoContext(classifierCtx, "Classifier result", "classifierResult", classifiedResult)
			classifierSpan.SetAttributes(keyClassifierResult.String(string(classifiedResult)))
			w.metricClassifierResult.Add(classifierCtx, 1, metric.WithAttributes(keyClassifierResult.String(string(classifiedResult))))
			w.metricClassifierDuration.Record(classifierCtx, finish.Seconds())
			classifierSpan.SetStatus(codes.Ok, "")

			// If the classifier reported one of the blocking result then abort generation early
			isBlocked := (w.config.Classifier.BlockNSFW && classifiedResult == ClassifierResultNsfw) ||
				(w.config.Classifier.BlockCSAM && classifiedResult == ClassifierResultCsam) ||
				(w.config.Classifier.FailClose && classifiedResult == ClassifierResultError)
			if isBlocked {
				w.logger.DebugContext(classifierCtx, "Aborting generation")
				generationCtxCancel(errContentClassifier)
				classifierSpan.SetAttributes(keyClassifierBlocked.Bool(true))
			} else {
				classifierSpan.SetAttributes(keyClassifierBlocked.Bool(false))
			}

			return nil
		})
	}

	err := errGroup.Wait()
	if err != nil {
		return fmt.Errorf("inference error: %w", err)
	}
	metadata := make([]aihorde.GenerationMetadataKobold, 0)
	state := aihorde.OptSubmitInputKoboldState{}

	// From talking to Horde developers, they seem to think that the classifier code path is basically untested
	// and the way to report statuses are varied
	switch classifiedResult {
	case ClassifierResultCsam:
		metadata = append(metadata, aihorde.GenerationMetadataKobold{
			Type:  aihorde.GenerationMetadataKoboldTypeCensorship,
			Value: aihorde.GenerationMetadataKoboldValueCsam,
		})
		state = aihorde.NewOptSubmitInputKoboldState(aihorde.SubmitInputKoboldStateCsam)

		// CSAM is always reported if classifier is run, but the generation will return unless the block is required
		if !w.config.Classifier.BlockCSAM {
			break
		}
		generation = "[Blocked by worker's content moderation]"
	case ClassifierResultNsfw:
		if !w.config.Classifier.BlockNSFW {
			break
		}
		state = aihorde.NewOptSubmitInputKoboldState(aihorde.SubmitInputKoboldStateCensored)
		generation = "[Blocked by worker's content moderation. This worker is SFW-only. Mark risky requests as NSFW to avoid such workers]"
	case ClassifierResultError:
		if !w.config.Classifier.FailClose {
			break
		}
		state = aihorde.NewOptSubmitInputKoboldState(aihorde.SubmitInputKoboldStateFaulted)
		generation = "[Worker's content moderation fault, and fail close policy is active]"
	}

	w.submitWg.Go(func() {
		// Don't use ctx here it will expire
		w.SubmitJobAsync(parentCtx, &aihorde.SubmitInputKobold{
			ID:          job.ID.Value,
			Generation:  generation,
			State:       state,
			GenMetadata: metadata,
		})
	})

	return nil
}

func (w *Worker) SubmitJob(ctx context.Context, result *aihorde.SubmitInputKobold) error {
	logger := w.logger.With("jobId", result.ID)
	logger.DebugContext(ctx, "Submitting job result")

	submitJobCtx, submitJobSpan := w.tracer.Start(ctx, "SubmitJob")
	defer submitJobSpan.End()

	submitRes, err := w.aihorde.PostTextJobSubmit(submitJobCtx, result, aihorde.PostTextJobSubmitParams{
		Apikey: w.config.HordeAPIKey,
	})

	if err != nil {
		logger.ErrorContext(ctx, "Failed to submit job", "err", err, "jobId", result.ID)
		submitJobSpan.RecordError(err)
		submitJobSpan.SetStatus(codes.Error, err.Error())
		return err
	}

	switch res := submitRes.(type) {
	case *aihorde.GenerationSubmitted:
		logger.InfoContext(submitJobCtx, "Job completed", "outputLen", len(result.Generation))
		w.metricKudos.Add(submitJobCtx, res.Reward.Value)
		submitJobSpan.SetStatus(codes.Ok, "")
		return nil
	default:
		err = fmt.Errorf("unknown submission response type %+v", submitRes)
		submitJobSpan.RecordError(err)
		submitJobSpan.SetStatus(codes.Error, err.Error())
		logger.WarnContext(submitJobCtx, "Unknown submission response type", "response", submitRes, "jobId", result.ID)
		return err
	}
}

func (w *Worker) SubmitJobAsync(ctx context.Context, result *aihorde.SubmitInputKobold) {
	w.submitWg.Go(func() {
		w.SubmitJob(ctx, result)
	})
}
