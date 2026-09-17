package usecase

import (
	"context"

	"go-feature-based-boilerplate/internal/auth/dto"
	"go-feature-based-boilerplate/internal/auth/repository"
	"go-feature-based-boilerplate/internal/auth/validator"
	"go-feature-based-boilerplate/pkg/cryptoutil"
)

// LogoutUsecase revokes a refresh token (single-session logout).
type LogoutUsecase struct {
	authRepo  repository.Repository
	validator *validator.Validator
}

// NewLogoutUsecase constructs a LogoutUsecase.
func NewLogoutUsecase(authRepo repository.Repository, v *validator.Validator) *LogoutUsecase {
	return &LogoutUsecase{authRepo: authRepo, validator: v}
}

// Execute revokes the provided refresh token.
func (u *LogoutUsecase) Execute(ctx context.Context, req dto.LogoutRequest) error {
	if err := u.validator.Validate(req); err != nil {
		return err
	}
	return u.authRepo.RevokeToken(ctx, cryptoutil.HashSHA256(req.RefreshToken))
}
