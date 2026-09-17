package postgres

import (
	"context"
	"errors"
	"strings"

	infraPostgres "go-feature-based-boilerplate/infrastructure/database/postgres"
	"go-feature-based-boilerplate/internal/user/entity"
	userErrors "go-feature-based-boilerplate/internal/user/errors"
	"go-feature-based-boilerplate/pkg/errorutil"

	"gorm.io/gorm"
)

// UserRepository implements user/repository.Repository against PostgreSQL via GORM.
type UserRepository struct {
	db *gorm.DB
}

// NewUserRepository creates a new PostgreSQL-backed user repository.
func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) getDB(ctx context.Context) *gorm.DB {
	return infraPostgres.GetDB(ctx, r.db)
}

func (r *UserRepository) FindByID(ctx context.Context, id uint) (*entity.User, error) {
	var u entity.User
	if err := r.getDB(ctx).First(&u, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, userErrors.ErrNotFound
		}
		return nil, errorutil.Wrap(errorutil.CodeInternal, "find user by id", err)
	}
	return &u, nil
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*entity.User, error) {
	var u entity.User
	if err := r.getDB(ctx).Where("email = ?", email).First(&u).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, userErrors.ErrNotFound
		}
		return nil, errorutil.Wrap(errorutil.CodeInternal, "find user by email", err)
	}
	return &u, nil
}

func (r *UserRepository) Create(ctx context.Context, u *entity.User) error {
	if err := r.getDB(ctx).Create(u).Error; err != nil {
		if isUniqueViolation(err) {
			return userErrors.ErrAlreadyExists
		}
		return errorutil.Wrap(errorutil.CodeInternal, "create user", err)
	}
	return nil
}

func (r *UserRepository) Update(ctx context.Context, u *entity.User) error {
	result := r.getDB(ctx).Save(u)
	if result.Error != nil {
		return errorutil.Wrap(errorutil.CodeInternal, "update user", result.Error)
	}
	if result.RowsAffected == 0 {
		return userErrors.ErrNotFound
	}
	return nil
}

func (r *UserRepository) Delete(ctx context.Context, id uint) error {
	result := r.getDB(ctx).Delete(&entity.User{}, id)
	if result.Error != nil {
		return errorutil.Wrap(errorutil.CodeInternal, "delete user", result.Error)
	}
	if result.RowsAffected == 0 {
		return userErrors.ErrNotFound
	}
	return nil
}

// isUniqueViolation reports whether the error is a PostgreSQL unique-constraint violation (code 23505).
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "23505") ||
		strings.Contains(err.Error(), "unique constraint") ||
		strings.Contains(err.Error(), "duplicate key")
}
