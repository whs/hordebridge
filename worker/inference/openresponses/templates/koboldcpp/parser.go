package koboldcpp

import (
	"fmt"
	"slices"
	"strings"

	"github.com/openai/openai-go/v3/packages/param"
	"github.com/openai/openai-go/v3/responses"
	"github.com/whs/hordebridge/aihorde"
	"github.com/whs/hordebridge/worker/inference/openresponses/templates"
)

// stopTags are potential stop tokens
var stopTags = []string{
	"{{[INPUT]}}",
	"{{[OUTPUT]}}",
	"{{[SYSTEM]}}",
	"### Instruction:",
	"### Response:",
	"<start_of_turn>user",
	"<start_of_turn>model",
	"<start_of_turn>system",
}

type Parser struct {
}

var _ templates.TemplateParser = Parser{}

func (k Parser) Parse(input *aihorde.ModelPayloadKobold) (responses.ResponseInputParam, error) {
	// Fast path: Stop sequence should have the special tags
	hasStopTag := slices.ContainsFunc(input.StopSequence, func(s string) bool {
		return slices.Contains(stopTags, strings.Trim(s, " \r\n"))
	})
	if !hasStopTag {
		return nil, templates.ErrTemplateNoMatch
	}

	matches, err := Parse("", []byte(input.Prompt.Value))
	if err != nil {
		// peg has no error types
		if strings.Contains(err.Error(), "no match found") {
			return nil, templates.ErrTemplateNoMatch
		}
		return nil, err
	}

	out := make(responses.ResponseInputParam, 0)

	var walk func(any) error
	var lastString strings.Builder
	hasTag := false

	flushLastString := func() error {
		if lastString.Len() == 0 {
			return nil
		}
		if len(out) == 0 {
			// If there is no message at all, then flush it as user message
			out = append(out, responses.ResponseInputItemUnionParam{
				OfMessage: &responses.EasyInputMessageParam{
					Role: responses.EasyInputMessageRoleUser,
					Type: responses.EasyInputMessageTypeMessage,
				},
			})
		}

		out[len(out)-1].OfMessage.Content.OfString = param.NewOpt(lastString.String())

		lastString.Reset()
		return nil
	}

	walk = func(node any) error {
		if node == any(nil) {
			return nil
		}

		switch n := node.(type) {
		case []any:
			for _, item := range n {
				err = walk(item)
				if err != nil {
					return err
				}
			}
			return nil
		case EndOfTurn:
			return nil
		case responses.EasyInputMessageRole:
			hasTag = true
			err = flushLastString()
			if err != nil {
				return err
			}
			out = append(out, responses.ResponseInputItemUnionParam{
				OfMessage: &responses.EasyInputMessageParam{
					Role: n,
					Type: responses.EasyInputMessageTypeMessage,
				},
			})
			return nil
		case []byte:
			lastString.Write(n)
			return nil
		default:
			return fmt.Errorf("unknown node type %T", n)
		}
	}

	err = walk(matches)
	if err != nil {
		return nil, err
	}
	if !hasTag {
		// If there is no tag then we just match the whole string for nothin
		return nil, templates.ErrTemplateNoMatch
	}
	err = flushLastString()

	// validate that the last message either be:
	// 1. assistant
	//    - if assistant has empty message, then we pop it out
	// 2. any role with non-empty message
	if len(out) > 0 {
		lastMessage := out[len(out)-1].OfMessage
		if lastMessage.Role == responses.EasyInputMessageRoleAssistant {
			if !lastMessage.Content.OfString.Valid() || len(lastMessage.Content.OfString.String()) == 0 {
				// pop last message
				out = out[:len(out)-1]
			}
		} else {
			if !lastMessage.Content.OfString.Valid() || len(lastMessage.Content.OfString.String()) == 0 {
				return nil, fmt.Errorf("last message must be non-empty: %w", templates.ErrTemplateNoMatch)
			}
		}
	}

	return out, err
}
