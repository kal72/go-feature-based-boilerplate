package repository

//go:generate mockgen -source=interface.go -destination=mock_repository.go -package=repository

import (
	"context"

	"go-feature-based-boilerplate/internal/auth/entity"
)

// Repository is the persistence contract for the auth feature.
type Repository interface {
	FindTokenByValue(ctx context.Context, token string) (*entity.RefreshToken, error)
	CreateToken(ctx context.Context, token *entity.RefreshToken) error
	RevokeToken(ctx context.Context, token string) error
	RevokeAllUserTokens(ctx context.Context, userID uint) error
}
