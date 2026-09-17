package usecase

import (
	"context"

	"go-feature-based-boilerplate/internal/user/dto"
	"go-feature-based-boilerplate/internal/user/entity"
	"go-feature-based-boilerplate/internal/user/repository"
	"go-feature-based-boilerplate/internal/user/validator"
	"go-feature-based-boilerplate/pkg/cryptoutil"
)

// CreateUsecase handles user creation.
type CreateUsecase struct {
	repo      repository.Repository
	validator *validator.Validator
}

// NewCreateUsecase constructs a CreateUsecase.
func NewCreateUsecase(repo repository.Repository, v *validator.Validator) *CreateUsecase {
	return &CreateUsecase{repo: repo, validator: v}
}

// Execute creates a new user and returns the persisted entity.
func (u *CreateUsecase) Execute(ctx context.Context, req dto.CreateUserRequest) (*entity.User, error) {
	if err := u.validator.Validate(req); err != nil {
		return nil, err
	}

	hashedPassword, err := cryptoutil.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	user := &entity.User{
		Name:     req.Name,
		Email:    req.Email,
		Password: hashedPassword,
		Role:     entity.RoleUser,
		Active:   true,
	}

	if err := u.repo.Create(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}
