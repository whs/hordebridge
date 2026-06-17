package inference

import (
	"context"
	"fmt"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/packages/param"
	"github.com/whs/hordebridge/aihorde"
)

type OpenAITextCompletion struct {
	client openai.Client
	config OpenAICompletionConfig
}

type OpenAICompletionConfig struct {
	Model string
}

var _ TextInference = &OpenAITextCompletion{}

func NewOpenAICompletion(client openai.Client, config OpenAICompletionConfig) TextInference {
	return &OpenAITextCompletion{
		client: client,
		config: config,
	}
}

func (o *OpenAITextCompletion) GenerateText(ctx context.Context, job *aihorde.GenerationPayloadKobold) (string, error) {
	payload, ok := job.Payload.Get()
	if !ok {
		return "", fmt.Errorf("no job payload")
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

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("openai returned no choices")
	}

	return resp.Choices[0].Text, nil
}
