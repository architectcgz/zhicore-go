package application

import "errors"

type Code string

const (
	CodeInvalidArgument Code = "INVALID_ARGUMENT"
)

type Error struct {
	Message string
	Code    Code
}

func (e *Error) Error() string {
	return e.Message
}

func invalidArgument(message string) error {
	return &Error{Code: CodeInvalidArgument, Message: message}
}

func AsError(err error) (*Error, bool) {
	var appErr *Error
	if errors.As(err, &appErr) {
		return appErr, true
	}
	return nil, false
}
