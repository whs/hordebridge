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

	out, err = templateParserKoboldCpp("Cont. prompt{{[INPUT]}}User prompt{{[OUTPUT]}}")
	assert.NoError(t, err)
	assert.Equal(t, responses.ResponseInputParam{
		{
			OfMessage: &responses.EasyInputMessageParam{
				Content: responses.EasyInputMessageContentUnionParam{
					OfString: param.NewOpt("Cont. prompt"),
				},
				Role: responses.EasyInputMessageRoleUser,
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

func TestParseKoboldEndTags(t *testing.T) {
	out, err := templateParserKoboldCpp("{{[SYSTEM]}}System prompt{{[SYSTEM_END]}}{{[INPUT]}}User prompt{{[INPUT_END]}}{{[OUTPUT]}}")
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
}

func TestParseKoboldAlpaca(t *testing.T) {
	out, err := templateParserKoboldCpp("System prompt\n### Instruction:\nUser prompt\n### Response:\n")
	assert.NoError(t, err)
	assert.Equal(t, responses.ResponseInputParam{
		{
			OfMessage: &responses.EasyInputMessageParam{
				Content: responses.EasyInputMessageContentUnionParam{
					OfString: param.NewOpt("System prompt"),
				},
				// TODO: Detect alpaca tag then set the initial prompt as system prompt
				Role: responses.EasyInputMessageRoleUser,
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

	out, err = templateParserKoboldCpp("\n### Instruction:\nUser prompt\n### Response:\nPrefill")
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

	out, err = templateParserKoboldCpp("### Instruction:\nUser prompt")
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
}

func TestParseKoboldGemma(t *testing.T) {
	out, err := templateParserKoboldCpp("<start_of_turn>system\nSystem prompt<end_of_turn>\n<start_of_turn>user\nUser prompt<end_of_turn>\n<start_of_turn>model\nModel")
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
					OfString: param.NewOpt("Model"),
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
