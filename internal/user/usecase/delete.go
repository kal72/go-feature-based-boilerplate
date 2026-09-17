package usecase

import (
	"context"

	userErrors "go-feature-based-boilerplate/internal/user/errors"
	"go-feature-based-boilerplate/internal/user/repository"
)

// DeleteUsecase handles user deletion.
type DeleteUsecase struct {
	repo repository.Repository
}

// NewDeleteUsecase constructs a DeleteUsecase.
func NewDeleteUsecase(repo repository.Repository) *DeleteUsecase {
	return &DeleteUsecase{repo: repo}
}

// Execute removes a user by ID if authorized.
func (u *DeleteUsecase) Execute(ctx context.Context, id uint, callerID uint) error {
	if callerID == 0 || callerID != id {
		return userErrors.ErrForbidden
	}
	return u.repo.Delete(ctx, id)
}
