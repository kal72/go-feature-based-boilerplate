package errors

import "go-feature-based-boilerplate/pkg/errorutil"

// Domain errors for the user feature.
var (
	ErrNotFound      = errorutil.New(errorutil.CodeNotFound, "user not found")
	ErrAlreadyExists = errorutil.New(errorutil.CodeAlreadyExists, "user already exists")
	ErrInvalidInput  = errorutil.New(errorutil.CodeInvalidInput, "invalid input")
	ErrForbidden     = errorutil.New(errorutil.CodeForbidden, "access denied")
)
