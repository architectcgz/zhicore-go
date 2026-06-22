package application

import "errors"

type Error struct {
	Status  int
	Message string
}

func (e *Error) Error() string {
	return e.Message
}

func errorf(status int, message string) error {
	return &Error{Status: status, Message: message}
}

func AsError(err error) (*Error, bool) {
	var appErr *Error
	if errors.As(err, &appErr) {
		return appErr, true
	}
	return nil, false
}
