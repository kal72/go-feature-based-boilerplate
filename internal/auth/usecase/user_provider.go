package usecase

import (
	"context"

	userEntity "go-feature-based-boilerplate/internal/user/entity"
)

// UserProvider defines the contract required by authentication use cases to query user details.
// Following the Interface Segregation Principle (ISP) and Ports & Adapters pattern,
// the Auth feature owns this port rather than coupling directly to the User repository package.
type UserProvider interface {
	FindByID(ctx context.Context, id uint) (*userEntity.User, error)
	FindByEmail(ctx context.Context, email string) (*userEntity.User, error)
}
