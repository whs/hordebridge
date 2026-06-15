package worker

import (
	"fmt"

	"github.com/go-faster/errors"
	"github.com/whs/hordebridge/aihorde"
)

type ReportableError struct {
	Kind          aihorde.SubmitInputKoboldState
	PublicError   string
	InternalError error
}

var _ error = ReportableError{}
var _ errors.Wrapper = ReportableError{}

func NewReportableError(err error, kind aihorde.SubmitInputKoboldState, message string, fmts ...any) ReportableError {
	return ReportableError{
		Kind:          kind,
		PublicError:   fmt.Sprintf(message, fmts...),
		InternalError: err,
	}
}

func (r ReportableError) Error() string {
	return r.InternalError.Error()
}

func (r ReportableError) Unwrap() error {
	return r.InternalError
}
