package errors

import "go-feature-based-boilerplate/pkg/errorutil"

// Domain errors for the auth feature.
var (
	ErrInvalidCredentials = errorutil.New(errorutil.CodeUnauthorized, "invalid email or password")
	ErrTokenNotFound      = errorutil.New(errorutil.CodeNotFound, "token not found")
	ErrTokenExpired       = errorutil.New(errorutil.CodeUnauthorized, "token has expired")
	ErrTokenRevoked       = errorutil.New(errorutil.CodeUnauthorized, "token has been revoked")
	ErrInvalidInput       = errorutil.New(errorutil.CodeInvalidInput, "invalid input")
)
