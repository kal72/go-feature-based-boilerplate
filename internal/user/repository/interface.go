package repository

//go:generate mockgen -source=interface.go -destination=mock_repository.go -package=repository

import (
	"context"

	"go-feature-based-boilerplate/internal/user/entity"
)

// Repository is the persistence contract for the user feature.
// Business logic depends only on this interface, never on the concrete
// infrastructure implementation.
type Repository interface {
	FindByID(ctx context.Context, id uint) (*entity.User, error)
	FindByEmail(ctx context.Context, email string) (*entity.User, error)
	Create(ctx context.Context, user *entity.User) error
	Update(ctx context.Context, user *entity.User) error
	Delete(ctx context.Context, id uint) error
}
