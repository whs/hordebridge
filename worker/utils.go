package worker

import "github.com/openai/openai-go/v3/packages/param"

type oasOpt[T any] interface {
	Get() (v T, ok bool)
}

func oasOptToOaiOpt[T comparable](val oasOpt[T]) param.Opt[T] {
	value, ok := val.Get()
	if !ok {
		return param.Null[T]()
	}

	return param.NewOpt(value)
}
