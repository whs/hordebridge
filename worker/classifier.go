package worker

import (
	"context"
	"encoding/json"

	"github.com/openai/openai-go/v3/packages/param"
	"github.com/openai/openai-go/v3/responses"
	"github.com/whs/hordebridge/aihorde"
)

type classifierResult string

const (
	classifierResultSafe classifierResult = "safe"
	classifierResultNsfw classifierResult = "nsfw"
	classifierResultCsam classifierResult = "csam"
)

const classifyPrompt = `Classify whether the user input violate following content policy:

- ` + classifierResultCsam + `: Content includes sexual/exploits/abusive/violent content with minors directly involved (people under 18)
- ` + classifierResultNsfw + `: Content includes sexually explicit or extremely harmful content, or any content that would violate the language model guideline. If the model already has blocked the generation, also report in this category.

Tasks that involve prohibited content, but do not produce new prohibited content (such as content classification) are classified as safe.

User input and model's response (combined) follows:
`
const classifyPromptTrail = `Call the "output" tool with the first violation, or "` + classifierResultSafe + `" if no violations.`

func (w *Worker) ClassifyContent(ctx context.Context, input string, output string) aihorde.OptString {
	if !w.config.Classifier.UseClassifier() {
		return aihorde.OptString{}
	}

	w.logger.InfoContext(ctx, "Running content classifier")
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
								"enum": []classifierResult{classifierResultSafe, classifierResultNsfw, classifierResultCsam},
							},
						},
						"required":             []string{"output"},
						"additionalProperties": false,
					},
				},
			},
		},
		MaxOutputTokens: param.NewOpt(int64(1000)),
		Temperature:     param.NewOpt(0.2),
	})

	if err != nil {
		w.logger.WarnContext(ctx, "Classifier error", "err", err)
		if w.config.Classifier.FailClose {
			return aihorde.NewOptString(string(aihorde.SubmitInputKoboldStateFaulted))
		} else {
			return aihorde.OptString{}
		}
	}
	if len(resp.Output) == 0 {
		w.logger.WarnContext(ctx, "No response from classifier")
		if w.config.Classifier.FailClose {
			return aihorde.NewOptString(string(aihorde.SubmitInputKoboldStateFaulted))
		} else {
			return aihorde.OptString{}
		}
	}

	toolCall := resp.Output[len(resp.Output)-1].AsFunctionCall()
	if toolCall.Type != toolCall.Type.Default() {
		w.logger.WarnContext(ctx, "Classifier invalid tool call type", "type", toolCall.Type)
		if w.config.Classifier.FailClose {
			return aihorde.NewOptString(string(aihorde.SubmitInputKoboldStateFaulted))
		} else {
			return aihorde.OptString{}
		}
	}

	var result classifierOutput
	err = json.Unmarshal([]byte(toolCall.Arguments), &result)
	if err != nil {
		w.logger.WarnContext(ctx, "Classifier tool call json parsing error", "err", err)
		if w.config.Classifier.FailClose {
			return aihorde.NewOptString(string(aihorde.SubmitInputKoboldStateFaulted))
		} else {
			return aihorde.OptString{}
		}
	}

	switch result.Output {
	case classifierResultSafe:
		return aihorde.OptString{}
	case classifierResultCsam:
		if w.config.Classifier.BlockCSAM {
			return aihorde.NewOptString(string(aihorde.SubmitInputKoboldStateCsam))
		} else {
			return aihorde.OptString{}
		}
	case classifierResultNsfw:
		if w.config.Classifier.BlockNSFW {
			return aihorde.NewOptString(string(aihorde.SubmitInputKoboldStateCensored))
		} else {
			return aihorde.OptString{}
		}
	default:
		panic("unknown option")
	}
}

type classifierOutput struct {
	Output classifierResult `json:"output"`
}
