package openresponses

import (
	"fmt"
	"strings"

	"github.com/go-faster/errors"
	"github.com/openai/openai-go/v3/packages/param"
	"github.com/openai/openai-go/v3/responses"
	"github.com/whs/hordebridge/worker/inference/openresponses/templates/koboldcpp"
)

type templateParser = func(input string) (responses.ResponseInputParam, error)

var ErrTemplateNoMatch = errors.New("template does not match input")

func templateParserKoboldCpp(input string) (responses.ResponseInputParam, error) {
	matches, err := koboldcpp.Parse("", []byte(input))
	if err != nil {
		// peg has no error types
		if strings.Contains(err.Error(), "no match found") {
			return nil, ErrTemplateNoMatch
		}
		return nil, err
	}

	out := make(responses.ResponseInputParam, 0)

	var walk func(any) error
	var lastString strings.Builder

	flushLastString := func() error {
		if lastString.Len() == 0 {
			return nil
		}
		if len(out) == 0 {
			return ErrTemplateNoMatch
		}

		out[len(out)-1].OfMessage.Content.OfString = param.NewOpt(lastString.String())

		lastString.Reset()
		return nil
	}

	walk = func(node any) error {
		switch n := node.(type) {
		case []any:
			for _, item := range n {
				err = walk(item)
				if err != nil {
					return err
				}
			}
			return nil
		case responses.EasyInputMessageRole:
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
				return nil, fmt.Errorf("last message must be non-empty: %w", ErrTemplateNoMatch)
			}
		}
	}

	return out, err
}

var _ templateParser = templateParserKoboldCpp
