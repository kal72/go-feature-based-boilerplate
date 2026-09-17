package errorutil

import "fmt"

// FieldError describes a validation failure on a single field.
type FieldError struct {
	Field   string
	Message string
}

// NewValidation creates an AppError for a field-level validation failure.
func NewValidation(field, message string) *AppError {
	return &AppError{
		Code:    CodeInvalidInput,
		Message: fmt.Sprintf("validation failed on field %s: %s", field, message),
	}
}
