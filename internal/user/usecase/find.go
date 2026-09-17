package usecase

import (
	"context"

	"go-feature-based-boilerplate/internal/user/entity"
	"go-feature-based-boilerplate/internal/user/repository"
)

// FindUsecase handles user retrieval.
type FindUsecase struct {
	repo repository.Repository
}

// NewFindUsecase constructs a FindUsecase.
func NewFindUsecase(repo repository.Repository) *FindUsecase {
	return &FindUsecase{repo: repo}
}

// FindByID retrieves a user by its primary key.
func (u *FindUsecase) FindByID(ctx context.Context, id uint) (*entity.User, error) {
	return u.repo.FindByID(ctx, id)
}

// FindByEmail retrieves a user by email address.
func (u *FindUsecase) FindByEmail(ctx context.Context, email string) (*entity.User, error) {
	return u.repo.FindByEmail(ctx, email)
}
