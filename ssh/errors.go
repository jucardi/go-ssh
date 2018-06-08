package ssh

import (
	"fmt"
	"github.com/getlantern/errors"
)

var (
	ErrNoConnection = errors.New("a client has not been established yet")
)

type ErrorWithExitCode struct {
	message  string
	ExitCode int
	Inner    error
}

func (e *ErrorWithExitCode) Error() string {
	return fmt.Sprintf("%s (%d) %v", e.message, e.ExitCode, e.Inner)
}

func makeErr(msg string, code int, inner ...error) *ErrorWithExitCode {
	var err error

	if len(inner) > 0 {
		err = inner[0]
	}

	return &ErrorWithExitCode{
		message:  msg,
		ExitCode: code,
		Inner:    err,
	}
}
