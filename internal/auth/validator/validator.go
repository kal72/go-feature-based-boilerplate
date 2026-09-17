package validator

import (
	"go-feature-based-boilerplate/pkg/errorutil"
	pkgvalidator "go-feature-based-boilerplate/pkg/validator"
)

// Validator wraps the shared validator for the auth feature.
type Validator struct{}

// New creates a Validator for the auth feature.
func New() *Validator { return &Validator{} }

// Validate validates the given struct and returns an AppError on failure.
func (v *Validator) Validate(s any) error {
	if err := pkgvalidator.ValidateStruct(s); err != nil {
		return errorutil.Wrap(errorutil.CodeInvalidInput, "validation failed", err)
	}
	return nil
}
