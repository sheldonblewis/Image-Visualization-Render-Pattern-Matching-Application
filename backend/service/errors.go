package service

import "fmt"

// ClientError represents a request validation issue that should surface as HTTP 400.
type ClientError struct {
	Msg string
}

func (e ClientError) Error() string { return e.Msg }

func newClientError(format string, args ...interface{}) error {
	return ClientError{Msg: fmt.Sprintf(format, args...)}
}

// IsClientError reports whether the error results from invalid user input.
func IsClientError(err error) bool {
	_, ok := err.(ClientError)
	return ok
}
