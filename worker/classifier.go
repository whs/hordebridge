package worker

import (
	"context"
	"encoding/json"

	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/packages/param"
	"github.com/openai/openai-go/v3/responses"
	"github.com/whs/hordebridge/worker/inference"
)

type ClassifierResult string

const (
	ClassifierResultSafe  ClassifierResult = "safe"
	ClassifierResultNsfw  ClassifierResult = "nsfw"
	ClassifierResultCsam  ClassifierResult = "csam"
	ClassifierResultError ClassifierResult = "error"
)

const classifyPrompt = `Quickly classify whether the user input violate following content policy:

- ` + ClassifierResultCsam + `: Content includes sexual/exploits/abusive/violent content with minors directly involved (people under 18)
- ` + ClassifierResultNsfw + `: Content includes sexually explicit or extremely harmful content, or any content that would violate the language model guideline. If the model already has blocked the generation, also report in this category.

Tasks that involve prohibited content, but do not produce new prohibited content (such as content classification) are classified as safe.

User input and model's response (combined) follows:
`
const classifyPromptTrail = `Call the "output" tool with the first violation, or "` + ClassifierResultSafe + `" if no violations.`

func (w *Worker) ClassifyContent(ctx context.Context, input string, output string) ClassifierResult {
	// XXX: In current version, output is always "" since it is run in parallel
	if !w.config.Classifier.UseClassifier() {
		return ClassifierResultSafe
	}

	additionalParams := make([]option.RequestOption, 0)
	if len(w.config.Classifier.AdditionalParams) > 0 {
		additionalParams = append(additionalParams, option.WithMiddleware(inference.JSONMergeMiddleware([]byte(w.config.Classifier.AdditionalParams))))
	}

	resp, err := w.openaiClassifier.Responses.New(ctx, responses.ResponseNewParams{
		Input: responses.ResponseNewParamsInputUnion{
			OfInputItemList: responses.ResponseInputParam{
				{
					OfMessage: &responses.EasyInputMessageParam{
						Role: responses.EasyInputMessageRoleSystem,
						Content: responses.EasyInputMessageContentUnionParam{
							OfString: param.NewOpt(string(classifyPrompt)),
						},
						Type: responses.EasyInputMessageTypeMessage,
					},
				},
				{
					OfMessage: &responses.EasyInputMessageParam{
						Role: responses.EasyInputMessageRoleUser,
						Content: responses.EasyInputMessageContentUnionParam{
							OfString: param.NewOpt(input + output),
						},
						Type: responses.EasyInputMessageTypeMessage,
					},
				},
				{
					OfMessage: &responses.EasyInputMessageParam{
						Role: responses.EasyInputMessageRoleSystem,
						Content: responses.EasyInputMessageContentUnionParam{
							OfString: param.NewOpt(string(classifyPromptTrail)),
						},
						Type: responses.EasyInputMessageTypeMessage,
					},
				},
			},
		},
		Model: w.config.Classifier.Model,
		ToolChoice: responses.ResponseNewParamsToolChoiceUnion{
			OfToolChoiceMode: param.NewOpt(responses.ToolChoiceOptionsRequired),
		},
		Tools: []responses.ToolUnionParam{
			{
				OfFunction: &responses.FunctionToolParam{
					Name: "output",
					Parameters: map[string]any{
						"type": "object",
						"properties": map[string]any{
							"output": map[string]any{
								"type": "string",
								"enum": []ClassifierResult{ClassifierResultSafe, ClassifierResultNsfw, ClassifierResultCsam},
							},
						},
						"required":             []string{"output"},
						"additionalProperties": false,
					},
				},
			},
		},
		MaxOutputTokens: param.NewOpt(w.config.Classifier.MaxTokens),
		Temperature:     param.NewOpt(w.config.Classifier.Temperature),
	}, additionalParams...)

	if err != nil {
		w.logger.WarnContext(ctx, "Classifier error", "err", err)
		if w.config.Classifier.FailClose {
			return ClassifierResultError
		} else {
			return ClassifierResultSafe
		}
	}
	if len(resp.Output) == 0 {
		w.logger.WarnContext(ctx, "No response from classifier")
		if w.config.Classifier.FailClose {
			return ClassifierResultError
		} else {
			return ClassifierResultSafe
		}
	}

	toolCall := resp.Output[len(resp.Output)-1].AsFunctionCall()
	if toolCall.Type != toolCall.Type.Default() {
		w.logger.WarnContext(ctx, "Classifier invalid tool call type", "type", toolCall.Type)
		if w.config.Classifier.FailClose {
			return ClassifierResultError
		} else {
			return ClassifierResultSafe
		}
	}

	var result classifierOutput
	err = json.Unmarshal([]byte(toolCall.Arguments), &result)
	if err != nil {
		w.logger.WarnContext(ctx, "Classifier tool call json parsing error", "err", err)
		if w.config.Classifier.FailClose {
			return ClassifierResultError
		} else {
			return ClassifierResultSafe
		}
	}

	return result.Output
}

type classifierOutput struct {
	Output ClassifierResult `json:"output"`
}
