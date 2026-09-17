package usecase

import (
	"context"

	"go-feature-based-boilerplate/internal/user/dto"
	"go-feature-based-boilerplate/internal/user/entity"
	userErrors "go-feature-based-boilerplate/internal/user/errors"
	"go-feature-based-boilerplate/internal/user/repository"
	"go-feature-based-boilerplate/internal/user/validator"
)

// UpdateUsecase handles user profile updates.
type UpdateUsecase struct {
	repo      repository.Repository
	validator *validator.Validator
}

// NewUpdateUsecase constructs an UpdateUsecase.
func NewUpdateUsecase(repo repository.Repository, v *validator.Validator) *UpdateUsecase {
	return &UpdateUsecase{repo: repo, validator: v}
}

// Execute applies updates to an existing user.
func (u *UpdateUsecase) Execute(ctx context.Context, req dto.UpdateUserRequest) (*entity.User, error) {
	if err := u.validator.Validate(req); err != nil {
		return nil, err
	}

	if req.CallerID == 0 || req.CallerID != req.ID {
		return nil, userErrors.ErrForbidden
	}

	user, err := u.repo.FindByID(ctx, req.ID)
	if err != nil {
		return nil, err
	}

	if req.Name != "" {
		user.Name = req.Name
	}

	if err := u.repo.Update(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}
