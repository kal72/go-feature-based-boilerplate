package errorutil

import "fmt"

// AppError is the standard application error. It carries a transport-agnostic
// Code so that infrastructure layers can map it to the appropriate transport
// status (gRPC code, HTTP status, etc.) in one central place.
type AppError struct {
	Code    Code
	Message string
	Err     error
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.Err
}

// New creates a new AppError without an underlying cause.
func New(code Code, message string) *AppError {
	return &AppError{Code: code, Message: message}
}

// Wrap creates an AppError that wraps an underlying error.
func Wrap(code Code, message string, err error) *AppError {
	return &AppError{Code: code, Message: message, Err: err}
}
