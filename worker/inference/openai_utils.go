package inference

import (
	"bytes"
	"io"
	"net/http"

	"github.com/evanphx/json-patch"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/packages/param"
)

type oasOpt[T any] interface {
	Get() (v T, ok bool)
}

func OasOptToOaiOpt[T comparable](val oasOpt[T]) param.Opt[T] {
	value, ok := val.Get()
	if !ok {
		return param.Null[T]()
	}

	return param.NewOpt(value)
}

type number interface {
	int | int8 | int16 | int32 | int64 | uint | uint8 | uint16 | uint32 | uint64 | float32 | float64
}

func OasOptCastToOaiOpt[I number, O number](val oasOpt[I]) param.Opt[O] {
	value, ok := val.Get()
	if !ok {
		return param.Null[O]()
	}

	return param.NewOpt(O(value))
}

func JSONMergeMiddleware(merge []byte) option.Middleware {
	return func(req *http.Request, next option.MiddlewareNext) (*http.Response, error) {
		oldBody, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		req.Body.Close()

		newBody, err := jsonpatch.MergePatch(oldBody, merge)
		if err != nil {
			return nil, err
		}

		req.Body = io.NopCloser(bytes.NewReader(newBody))
		req.ContentLength = int64(len(newBody))

		return next(req)
	}
}
