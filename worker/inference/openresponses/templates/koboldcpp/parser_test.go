package koboldcpp

import (
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/openai/openai-go/v3/packages/param"
	"github.com/openai/openai-go/v3/responses"
	"github.com/whs/hordebridge/aihorde"
	"github.com/whs/hordebridge/worker/inference/openresponses/templates"
)

func TestParseKobold(t *testing.T) {
	parser := Parser{}

	out, err := parser.Parse(&aihorde.ModelPayloadKobold{
		Prompt:       aihorde.NewOptString("{{[SYSTEM]}}System prompt{{[INPUT]}}User prompt{{[OUTPUT]}}"),
		StopSequence: []string{"{{[SYSTEM]}}"},
	})
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

	out, err = parser.Parse(&aihorde.ModelPayloadKobold{
		Prompt:       aihorde.NewOptString("Cont. prompt{{[INPUT]}}User prompt{{[OUTPUT]}}"),
		StopSequence: []string{"{{[SYSTEM]}}"},
	})
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

	out, err = parser.Parse(&aihorde.ModelPayloadKobold{
		Prompt:       aihorde.NewOptString("{{[INPUT]}}User prompt{{[OUTPUT]}}"),
		StopSequence: []string{"{{[SYSTEM]}}"},
	})
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

	out, err = parser.Parse(&aihorde.ModelPayloadKobold{
		Prompt:       aihorde.NewOptString("{{[INPUT]}}User prompt"),
		StopSequence: []string{"{{[SYSTEM]}}"},
	})
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

	out, err = parser.Parse(&aihorde.ModelPayloadKobold{
		Prompt:       aihorde.NewOptString("{{[SYSTEM]}}System prompt{{[INPUT]}}User prompt{{[OUTPUT]}}Prefill"),
		StopSequence: []string{"{{[SYSTEM]}}"},
	})
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
	parser := Parser{}

	out, err := parser.Parse(&aihorde.ModelPayloadKobold{
		Prompt:       aihorde.NewOptString("{{[SYSTEM]}}System prompt{{[SYSTEM_END]}}{{[INPUT]}}User prompt{{[INPUT_END]}}{{[OUTPUT]}}"),
		StopSequence: []string{"{{[SYSTEM]}}"},
	})
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
	parser := Parser{}

	out, err := parser.Parse(&aihorde.ModelPayloadKobold{
		Prompt:       aihorde.NewOptString("System prompt\n### Instruction:\nUser prompt\n### Response:\n"),
		StopSequence: []string{"### Instruction:\n"},
	})
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

	out, err = parser.Parse(&aihorde.ModelPayloadKobold{
		Prompt:       aihorde.NewOptString("\n### Instruction:\nUser prompt\n### Response:\nPrefill"),
		StopSequence: []string{"\n### Instruction:\n"},
	})
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

	out, err = parser.Parse(&aihorde.ModelPayloadKobold{
		Prompt:       aihorde.NewOptString("### Instruction:\nUser prompt"),
		StopSequence: []string{"### Instruction:\n"},
	})
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
	parser := Parser{}

	out, err := parser.Parse(&aihorde.ModelPayloadKobold{
		Prompt:       aihorde.NewOptString("<start_of_turn>system\nSystem prompt<end_of_turn>\n<start_of_turn>user\nUser prompt<end_of_turn>\n<start_of_turn>model\nModel"),
		StopSequence: []string{"<start_of_turn>system\n"},
	})
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
	parser := Parser{}

	_, err := parser.Parse(&aihorde.ModelPayloadKobold{
		Prompt:       aihorde.NewOptString("User: Hello\nAssistant:"),
		StopSequence: []string{"User: ", "Assistant: "},
	})
	assert.IsError(t, err, templates.ErrTemplateNoMatch)

	_, err = parser.Parse(&aihorde.ModelPayloadKobold{
		Prompt:       aihorde.NewOptString("{{[INPUT]}}User prompt{{[SYSTEM]}}"),
		StopSequence: []string{"{{[INPUT]}}"},
	})
	assert.IsError(t, err, templates.ErrTemplateNoMatch)

	_, err = parser.Parse(&aihorde.ModelPayloadKobold{
		Prompt:       aihorde.NewOptString("{{[INPUT]}}User prompt{{[INPUT]}}"),
		StopSequence: []string{"{{[INPUT]}}"},
	})
	assert.IsError(t, err, templates.ErrTemplateNoMatch)
}
