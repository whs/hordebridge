package openresponses

import (
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/openai/openai-go/v3/packages/param"
	"github.com/openai/openai-go/v3/responses"
)

func TestParseKobold(t *testing.T) {
	out, err := templateParserKoboldCpp("{{[SYSTEM]}}System prompt{{[INPUT]}}User prompt{{[OUTPUT]}}")
	assert.NoError(t, err)
	assert.Equal(t, responses.ResponseInputParam{
		{
			OfMessage: &responses.EasyInputMessageParam{
				Content: responses.EasyInputMessageContentUnionParam{
					OfString: param.NewOpt("System prompt"),
				},
				Role: responses.EasyInputMessageRoleSystem,
				Type: responses.EasyInputMessageTypeMessage,
			},
		},
		{
			OfMessage: &responses.EasyInputMessageParam{
				Content: responses.EasyInputMessageContentUnionParam{
					OfString: param.NewOpt("User prompt"),
				},
				Role: responses.EasyInputMessageRoleUser,
				Type: responses.EasyInputMessageTypeMessage,
			},
		},
	}, out)

	out, err = templateParserKoboldCpp("{{[INPUT]}}User prompt{{[OUTPUT]}}")
	assert.NoError(t, err)
	assert.Equal(t, responses.ResponseInputParam{
		{
			OfMessage: &responses.EasyInputMessageParam{
				Content: responses.EasyInputMessageContentUnionParam{
					OfString: param.NewOpt("User prompt"),
				},
				Role: responses.EasyInputMessageRoleUser,
				Type: responses.EasyInputMessageTypeMessage,
			},
		},
	}, out)

	out, err = templateParserKoboldCpp("{{[INPUT]}}User prompt")
	assert.NoError(t, err)
	assert.Equal(t, responses.ResponseInputParam{
		{
			OfMessage: &responses.EasyInputMessageParam{
				Content: responses.EasyInputMessageContentUnionParam{
					OfString: param.NewOpt("User prompt"),
				},
				Role: responses.EasyInputMessageRoleUser,
				Type: responses.EasyInputMessageTypeMessage,
			},
		},
	}, out)

	out, err = templateParserKoboldCpp("{{[SYSTEM]}}System prompt{{[INPUT]}}User prompt{{[OUTPUT]}}Prefill")
	assert.NoError(t, err)
	assert.Equal(t, responses.ResponseInputParam{
		{
			OfMessage: &responses.EasyInputMessageParam{
				Content: responses.EasyInputMessageContentUnionParam{
					OfString: param.NewOpt("System prompt"),
				},
				Role: responses.EasyInputMessageRoleSystem,
				Type: responses.EasyInputMessageTypeMessage,
			},
		},
		{
			OfMessage: &responses.EasyInputMessageParam{
				Content: responses.EasyInputMessageContentUnionParam{
					OfString: param.NewOpt("User prompt"),
				},
				Role: responses.EasyInputMessageRoleUser,
				Type: responses.EasyInputMessageTypeMessage,
			},
		},
		{
			OfMessage: &responses.EasyInputMessageParam{
				Content: responses.EasyInputMessageContentUnionParam{
					OfString: param.NewOpt("Prefill"),
				},
				Role: responses.EasyInputMessageRoleAssistant,
				Type: responses.EasyInputMessageTypeMessage,
			},
		},
	}, out)
}

func TestParseKoboldInvalid(t *testing.T) {
	_, err := templateParserKoboldCpp("User: Hello\nAssistant:")
	assert.IsError(t, err, ErrTemplateNoMatch)

	_, err = templateParserKoboldCpp("{{[INPUT]}}User prompt{{[SYSTEM]}}")
	assert.IsError(t, err, ErrTemplateNoMatch)

	_, err = templateParserKoboldCpp("{{[INPUT]}}User prompt{{[INPUT]}}")
	assert.IsError(t, err, ErrTemplateNoMatch)
}
