package postgres

import (
	"context"
	"errors"

	infraPostgres "go-feature-based-boilerplate/infrastructure/database/postgres"
	"go-feature-based-boilerplate/internal/auth/entity"
	authErrors "go-feature-based-boilerplate/internal/auth/errors"
	"go-feature-based-boilerplate/pkg/errorutil"

	"gorm.io/gorm"
)

// AuthRepository implements auth/repository.Repository against PostgreSQL via GORM.
type AuthRepository struct {
	db *gorm.DB
}

// NewAuthRepository creates a new PostgreSQL-backed auth repository.
func NewAuthRepository(db *gorm.DB) *AuthRepository {
	return &AuthRepository{db: db}
}

func (r *AuthRepository) getDB(ctx context.Context) *gorm.DB {
	return infraPostgres.GetDB(ctx, r.db)
}

func (r *AuthRepository) FindTokenByValue(ctx context.Context, token string) (*entity.RefreshToken, error) {
	var t entity.RefreshToken
	if err := r.getDB(ctx).Where("token = ?", token).First(&t).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, authErrors.ErrTokenNotFound
		}
		return nil, errorutil.Wrap(errorutil.CodeInternal, "find token", err)
	}
	return &t, nil
}

func (r *AuthRepository) CreateToken(ctx context.Context, t *entity.RefreshToken) error {
	if err := r.getDB(ctx).Create(t).Error; err != nil {
		return errorutil.Wrap(errorutil.CodeInternal, "create token", err)
	}
	return nil
}

func (r *AuthRepository) RevokeToken(ctx context.Context, token string) error {
	result := r.getDB(ctx).
		Model(&entity.RefreshToken{}).
		Where("token = ?", token).
		Update("revoked", true)
	if result.Error != nil {
		return errorutil.Wrap(errorutil.CodeInternal, "revoke token", result.Error)
	}
	if result.RowsAffected == 0 {
		return authErrors.ErrTokenNotFound
	}
	return nil
}

func (r *AuthRepository) RevokeAllUserTokens(ctx context.Context, userID uint) error {
	if err := r.getDB(ctx).
		Model(&entity.RefreshToken{}).
		Where("user_id = ? AND revoked = false", userID).
		Update("revoked", true).Error; err != nil {
		return errorutil.Wrap(errorutil.CodeInternal, "revoke all user tokens", err)
	}
	return nil
}
