package templates

import (
	"github.com/go-faster/errors"
	"github.com/openai/openai-go/v3/responses"
	"github.com/whs/hordebridge/aihorde"
)

type TemplateParser interface {
	Parse(input *aihorde.ModelPayloadKobold) (responses.ResponseInputParam, error)
}

var ErrTemplateNoMatch = errors.New("template does not match input")
