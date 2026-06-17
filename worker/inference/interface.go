package inference

import (
	"context"

	"github.com/whs/hordebridge/aihorde"
)

type TextInference interface {
	GenerateText(ctx context.Context, job *aihorde.GenerationPayloadKobold) (string, error)
}

type ErrorTextInference struct {
	Error error
}

var _ TextInference = ErrorTextInference{}

func (e ErrorTextInference) GenerateText(ctx context.Context, job *aihorde.GenerationPayloadKobold) (string, error) {
	return "", e.Error
}
