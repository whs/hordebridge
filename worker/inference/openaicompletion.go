package inference

import (
	"context"
	"fmt"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/packages/param"
	"github.com/whs/hordebridge/aihorde"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/semconv/v1.41.0/genaiconv"
)

type OpenAITextCompletion struct {
	client openai.Client
	config OpenAICompletionConfig

	metricTokenUsage genaiconv.ClientTokenUsage
}

type OpenAICompletionConfig struct {
	Model            string
	AdditionalParams []byte
}

var _ TextInference = &OpenAITextCompletion{}

func NewOpenAICompletion(client openai.Client, config OpenAICompletionConfig) (TextInference, error) {
	meter := otel.Meter("whs.in.th/hordebridge")
	metricTokenUsage, err := genaiconv.NewClientTokenUsage(meter)
	if err != nil {
		return nil, err
	}

	return &OpenAITextCompletion{
		client:           client,
		config:           config,
		metricTokenUsage: metricTokenUsage,
	}, nil
}

func (o *OpenAITextCompletion) GenerateText(ctx context.Context, job *aihorde.GenerationPayloadKobold) (string, error) {
	payload, ok := job.Payload.Get()
	if !ok {
		return "", fmt.Errorf("no job payload")
	}

	additionalParams := make([]option.RequestOption, 0)
	// https://github.com/Haidra-Org/AI-Horde-Worker/blob/d0bacd83f996550d934c105ea25ac7a0e0fb380e/worker/jobs/scribe.py#L78
	if repPen, ok := payload.RepPen.Get(); ok {
		additionalParams = append(additionalParams, option.WithJSONSet("repetition_penalty", repPen))
	}
	if dynatempExponent, ok := payload.DynatempExponent.Get(); ok {
		additionalParams = append(additionalParams, option.WithJSONSet("dynatemp_exponent", dynatempExponent))
	}
	if dynatempRange, ok := payload.DynatempRange.Get(); ok {
		additionalParams = append(additionalParams, option.WithJSONSet("dynatemp_range", dynatempRange))
	}
	if tfs, ok := payload.Tfs.Get(); ok {
		additionalParams = append(additionalParams, option.WithJSONSet("tfs", tfs))
	}
	if topK, ok := payload.TopK.Get(); ok && topK != 0.0 {
		additionalParams = append(additionalParams, option.WithJSONSet("top_k", topK))
	}
	if topA, ok := payload.TopA.Get(); ok {
		additionalParams = append(additionalParams, option.WithJSONSet("top_a", topA))
	}
	if minP, ok := payload.MinP.Get(); ok {
		additionalParams = append(additionalParams, option.WithJSONSet("min_p", minP))
	}
	if typical, ok := payload.Typical.Get(); ok {
		additionalParams = append(additionalParams, option.WithJSONSet("typical_p", typical))
	}
	if len(o.config.AdditionalParams) > 0 {
		additionalParams = append(additionalParams, option.WithMiddleware(JSONMergeMiddleware(o.config.AdditionalParams)))
	}

	resp, err := o.client.Completions.New(ctx, openai.CompletionNewParams{
		Prompt: openai.CompletionNewParamsPromptUnion{
			OfString: param.NewOpt(payload.Prompt.Value),
		},
		Model:       openai.CompletionNewParamsModel(o.config.Model),
		MaxTokens:   OasOptCastToOaiOpt[int, int64](payload.MaxLength),
		Temperature: OasOptToOaiOpt[float64](payload.Temperature),
		TopP:        OasOptToOaiOpt[float64](payload.TopP),
		Stop: openai.CompletionNewParamsStopUnion{
			OfStringArray: payload.StopSequence,
		},
	}, additionalParams...)

	if err != nil {
		return "", fmt.Errorf("openai error: %w", err)
	}

	o.metricTokenUsage.Record(ctx, resp.Usage.PromptTokens, genaiconv.OperationNameTextCompletion, "", genaiconv.TokenTypeInput, o.metricTokenUsage.AttrRequestModel(o.config.Model), o.metricTokenUsage.AttrResponseModel(resp.Model))
	o.metricTokenUsage.Record(ctx, resp.Usage.CompletionTokens, genaiconv.OperationNameTextCompletion, "", genaiconv.TokenTypeOutput, o.metricTokenUsage.AttrRequestModel(o.config.Model), o.metricTokenUsage.AttrResponseModel(resp.Model))

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("openai returned no choices")
	}

	return resp.Choices[0].Text, nil
}
