package openresponses

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/go-faster/errors"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/responses"
	"github.com/whs/hordebridge/aihorde"
	"github.com/whs/hordebridge/worker/inference"
	"github.com/whs/hordebridge/worker/inference/openresponses/templates"
	"github.com/whs/hordebridge/worker/inference/openresponses/templates/koboldcpp"
)

type OpenResponsesCompletion struct {
	client openai.Client
	config ResponsesConfig
	logger *slog.Logger
}

type ResponsesConfig struct {
	Model            string
	AdditionalParams []byte
	Fallback         inference.TextInference
	Parsers          []templates.TemplateParser
}

var _ inference.TextInference = &OpenResponsesCompletion{}

func New(client openai.Client, config ResponsesConfig) inference.TextInference {
	if len(config.Parsers) == 0 {
		config.Parsers = []templates.TemplateParser{
			koboldcpp.Parser{},
		}
	}

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

	var parsed responses.ResponseInputParam
	var err error
	for _, parser := range o.config.Parsers {
		parsed, err = parser.Parse(&payload)
		if errors.Is(err, templates.ErrTemplateNoMatch) {
			// XXX: Don't forget to preserve last err in this block!
			continue
		} else if err != nil {
			return "", fmt.Errorf("chat template execution failed: %w", err)
		}
		break
	}
	if errors.Is(err, templates.ErrTemplateNoMatch) {
		// Fallback when the chat template doesn't match
		return o.config.Fallback.GenerateText(ctx, job)
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
		additionalParams = append(additionalParams, option.WithMiddleware(inference.JSONMergeMiddleware(o.config.AdditionalParams)))
	}

	o.logger.DebugContext(ctx, "Using responses API", "conversation_length", len(parsed), "last_turn_role", parsed[len(parsed)-1].OfMessage.Role)
	resp, err := o.client.Responses.New(ctx, responses.ResponseNewParams{
		MaxOutputTokens: inference.OasOptCastToOaiOpt[int, int64](payload.MaxLength),
		Temperature:     inference.OasOptToOaiOpt(payload.Temperature),
		TopP:            inference.OasOptToOaiOpt(payload.TopP),
		Input: responses.ResponseNewParamsInputUnion{
			OfInputItemList: parsed,
		},
		Model: o.config.Model,
	}, additionalParams...)

	if err != nil {
		return "", fmt.Errorf("openai error: %w", err)
	}

	out := resp.OutputText()

	// If prefill is used, then strip the prefill
	parsed[len(parsed)-1].GetRole()
	usedPrefill := parsed[len(parsed)-1].OfMessage.Role == responses.EasyInputMessageRoleAssistant
	if usedPrefill {
		prefill := parsed[len(parsed)-1].OfMessage.Content.OfString.String()
		out = strings.TrimPrefix(out, prefill)
	}

	return out, nil
}
