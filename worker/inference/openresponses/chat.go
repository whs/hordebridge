package openresponses

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/go-faster/errors"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/responses"
	"github.com/whs/hordebridge/aihorde"
	"github.com/whs/hordebridge/worker/inference"
)

type OpenResponsesCompletion struct {
	client openai.Client
	config ResponsesConfig
	logger *slog.Logger
}

type ResponsesConfig struct {
	Model    string
	Fallback inference.TextInference
}

var _ inference.TextInference = &OpenResponsesCompletion{}

func New(client openai.Client, config ResponsesConfig) inference.TextInference {
	return &OpenResponsesCompletion{
		client: client,
		config: config,
		logger: slog.Default().With("module", "openresponses"),
	}
}

func (o *OpenResponsesCompletion) GenerateText(ctx context.Context, job *aihorde.GenerationPayloadKobold) (string, error) {
	payload, ok := job.Payload.Get()
	if !ok {
		return "", fmt.Errorf("no job payload")
	}

	parsed, err := templateParserKoboldCpp(payload.Prompt.Value)
	if errors.Is(err, ErrTemplateNoMatch) {
		// Fallback when the chat template doesn't match
		return o.config.Fallback.GenerateText(ctx, job)
	} else if err != nil {
		return "", fmt.Errorf("chat template execution failed: %w", err)
	}

	o.logger.DebugContext(ctx, "Using responses API", "conversation_length", len(parsed), "last_turn_role", parsed[len(parsed)-1].OfMessage.Role)
	resp, err := o.client.Responses.New(ctx, responses.ResponseNewParams{
		MaxOutputTokens: inference.OasOptCastToOaiOpt[int, int64](payload.MaxLength),
		Temperature:     inference.OasOptToOaiOpt[float64](payload.Temperature),
		TopP:            inference.OasOptToOaiOpt[float64](payload.TopP),
		Input: responses.ResponseNewParamsInputUnion{
			OfInputItemList: parsed,
		},
		Model: o.config.Model,
	})

	if err != nil {
		return "", fmt.Errorf("openai error: %w", err)
	}

	return resp.OutputText(), nil
}
