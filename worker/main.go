package worker

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/go-faster/errors"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/packages/param"
	"github.com/whs/hordebridge/aihorde"
)

type Worker struct {
	config  Config
	logger  *slog.Logger
	aihorde *aihorde.Client
	openai  openai.Client
}

func NewWorker(config Config) (*Worker, error) {
	aihordeClient, err := aihorde.NewClient(config.HordeServer)
	if err != nil {
		return nil, err
	}
	return &Worker{
		config:  config,
		logger:  slog.Default().With("module", "worker"),
		aihorde: aihordeClient,
		openai:  openai.NewClient(option.WithAPIKey(config.OpenaiAPIKey), option.WithBaseURL(config.OpenaiServer)),
	}, nil
}

// Start the main worker loop
// The worker loop is defined in https://github.com/Haidra-Org/haidra-assets/blob/main/docs/workers.md
func (w *Worker) Start(ctx context.Context, abortCtx context.Context) {
	waitTime := 0 * time.Second
	errorCount := 0

	sleep := func(dur time.Duration) {
		select {
		case <-ctx.Done():
		case <-time.After(dur):
		}
	}

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
			waitTime = min(3*time.Second, 500*time.Millisecond+waitTime)
			w.logger.DebugContext(ctx, "No job available", "wait", waitTime)
			sleep(waitTime)
			continue
		}

		// Got a job!
		waitTime = 0
		err = w.ProcessJob(ctx, job)
		if err != nil {
			w.logger.ErrorContext(ctx, "Failed to process job. Sending error", "err", err)

			reportable, ok := errors.Into[ReportableError](err)
			// XXX: Use abortCtx to ensure that if ctx is canceled, this job should be able to send the report
			if ok {
				_, sendErrErr := w.aihorde.PostTextJobSubmit(abortCtx, &aihorde.SubmitInputKobold{
					ID:         job.ID.Value,
					Generation: reportable.PublicError,
					State:      aihorde.NewOptSubmitInputKoboldState(reportable.Kind),
				}, aihorde.PostTextJobSubmitParams{
					Apikey: w.config.HordeAPIKey,
				})
				if sendErrErr != nil {
					w.logger.ErrorContext(ctx, "Failed to send job error. Exiting", "err", sendErrErr)
					return
				}
			} else {
				_, sendErrErr := w.aihorde.PostTextJobSubmit(abortCtx, &aihorde.SubmitInputKobold{
					ID:    job.ID.Value,
					State: aihorde.NewOptSubmitInputKoboldState(aihorde.SubmitInputKoboldStateFaulted),
				}, aihorde.PostTextJobSubmitParams{
					Apikey: w.config.HordeAPIKey,
				})
				if sendErrErr != nil {
					w.logger.ErrorContext(ctx, "Failed to send job error. Exiting", "err", sendErrErr)
					return
				}
			}

			if onError(err) {
				return
			}
			continue
		}

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
	logger.InfoContext(parentCtx, "Processing job")

	ctx, cancel := context.WithTimeout(parentCtx, time.Duration(job.TTL.Or(60*60))*time.Second)
	defer cancel()

	payload, ok := job.Payload.Get()
	if !ok {
		return fmt.Errorf("no job payload")
	}

	// TODO: Don't silently truncate maxToken
	maxTokens := int64(payload.MaxLength.Or(w.config.MaxLength))
	if maxTokens > int64(w.config.MaxLength) {
		return NewReportableError(errors.New("max_length validation error"), aihorde.SubmitInputKoboldStateFaulted, "Requested max length %d > allowed %d", maxTokens, w.config.MaxLength)
	}

	additionalParams := make([]option.RequestOption, 0)
	if topK, ok := payload.TopK.Get(); ok {
		additionalParams = append(additionalParams, option.WithJSONSet("top_k", topK))
	}
	if minP, ok := payload.MinP.Get(); ok {
		additionalParams = append(additionalParams, option.WithJSONSet("min_p", minP))
	}
	if typical, ok := payload.Typical.Get(); ok {
		additionalParams = append(additionalParams, option.WithJSONSet("typical_p", typical))
	}
	if repPen, ok := payload.RepPen.Get(); ok {
		additionalParams = append(additionalParams, option.WithJSONSet("repetition_penalty", repPen))
	}

	resp, err := w.openai.Completions.New(ctx, openai.CompletionNewParams{
		Prompt: openai.CompletionNewParamsPromptUnion{
			OfString: param.NewOpt(payload.Prompt.Value),
		},
		Model:       openai.CompletionNewParamsModel(w.config.OpenaiModel),
		MaxTokens:   param.NewOpt(maxTokens),
		Temperature: oasOptToOaiOpt[float64](payload.Temperature),
		TopP:        oasOptToOaiOpt[float64](payload.TopP),
		Stop: openai.CompletionNewParamsStopUnion{
			OfStringArray: payload.StopSequence,
		},
	}, additionalParams...)

	if err != nil {
		return fmt.Errorf("openai error: %w", err)
	}

	if len(resp.Choices) == 0 {
		return fmt.Errorf("openai returned no choices")
	}

	logger.InfoContext(ctx, "Sending job result", "length", len(resp.Choices[0].Text))
	submitRes, err := w.aihorde.PostTextJobSubmit(ctx, &aihorde.SubmitInputKobold{
		ID:          job.ID.Value,
		Generation:  resp.Choices[0].Text,
		State:       aihorde.NewOptSubmitInputKoboldState(aihorde.SubmitInputKoboldStateOk),
		GenMetadata: nil,
	}, aihorde.PostTextJobSubmitParams{
		Apikey: w.config.HordeAPIKey,
	})

	if err != nil {
		return fmt.Errorf("failed to submit job: %w", err)
	}

	switch submitRes.(type) {
	case *aihorde.GenerationSubmitted:
		logger.InfoContext(ctx, "Job completed")
		return nil
	default:
		return fmt.Errorf("unknown response type: %+v", submitRes)
	}
}
