package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"go-feature-based-boilerplate/internal/auth/dto"
	authEntity "go-feature-based-boilerplate/internal/auth/entity"
	authErrors "go-feature-based-boilerplate/internal/auth/errors"
	"go-feature-based-boilerplate/internal/auth/usecase"
	"go-feature-based-boilerplate/internal/auth/validator"
	userEntity "go-feature-based-boilerplate/internal/user/entity"
	"go-feature-based-boilerplate/pkg/cryptoutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockAuthRepository struct {
	mock.Mock
}

func (m *mockAuthRepository) FindTokenByValue(ctx context.Context, token string) (*authEntity.RefreshToken, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*authEntity.RefreshToken), args.Error(1)
}

func (m *mockAuthRepository) CreateToken(ctx context.Context, token *authEntity.RefreshToken) error {
	args := m.Called(ctx, token)
	return args.Error(0)
}

func (m *mockAuthRepository) RevokeToken(ctx context.Context, token string) error {
	args := m.Called(ctx, token)
	return args.Error(0)
}

func (m *mockAuthRepository) RevokeAllUserTokens(ctx context.Context, userID uint) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

type mockUserProvider struct {
	mock.Mock
}

func (m *mockUserProvider) FindByID(ctx context.Context, id uint) (*userEntity.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*userEntity.User), args.Error(1)
}

func (m *mockUserProvider) FindByEmail(ctx context.Context, email string) (*userEntity.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*userEntity.User), args.Error(1)
}

func testTokenConfig() usecase.TokenConfig {
	return usecase.TokenConfig{
		SecretKey:  "test-secret-key-that-is-long-enough",
		AccessTTL:  15 * time.Minute,
		RefreshTTL: 24 * time.Hour,
	}
}

func TestLoginUsecase_Success(t *testing.T) {
	authRepo := new(mockAuthRepository)
	userProv := new(mockUserProvider)
	v := validator.New()
	uc := usecase.NewLoginUsecase(authRepo, userProv, v, testTokenConfig())

	hashedPassword, err := cryptoutil.HashPassword("secret123")
	assert.NoError(t, err)

	u := &userEntity.User{
		ID:       1,
		Email:    "alice@example.com",
		Password: hashedPassword,
		Role:     userEntity.RoleUser,
		Active:   true,
	}

	userProv.On("FindByEmail", mock.Anything, "alice@example.com").Return(u, nil)
	authRepo.On("CreateToken", mock.Anything, mock.MatchedBy(func(t *authEntity.RefreshToken) bool {
		return t.UserID == 1
	})).Return(nil)

	pair, err := uc.Execute(context.Background(), dto.LoginRequest{
		Email:    "alice@example.com",
		Password: "secret123",
	})

	assert.NoError(t, err)
	assert.NotNil(t, pair)
	assert.NotEmpty(t, pair.AccessToken)
	assert.NotEmpty(t, pair.RefreshToken)
	authRepo.AssertExpectations(t)
	userProv.AssertExpectations(t)
}

func TestLoginUsecase_UserNotFound(t *testing.T) {
	authRepo := new(mockAuthRepository)
	userProv := new(mockUserProvider)
	v := validator.New()
	uc := usecase.NewLoginUsecase(authRepo, userProv, v, testTokenConfig())

	userProv.On("FindByEmail", mock.Anything, "nonexistent@example.com").Return(nil, errors.New("not found"))

	pair, err := uc.Execute(context.Background(), dto.LoginRequest{
		Email:    "nonexistent@example.com",
		Password: "secret123",
	})

	assert.Nil(t, pair)
	assert.ErrorIs(t, err, authErrors.ErrInvalidCredentials)
}

func TestLoginUsecase_WrongPassword(t *testing.T) {
	authRepo := new(mockAuthRepository)
	userProv := new(mockUserProvider)
	v := validator.New()
	uc := usecase.NewLoginUsecase(authRepo, userProv, v, testTokenConfig())

	hashedPassword, err := cryptoutil.HashPassword("secret123")
	assert.NoError(t, err)

	u := &userEntity.User{
		ID:       1,
		Email:    "alice@example.com",
		Password: hashedPassword,
	}

	userProv.On("FindByEmail", mock.Anything, "alice@example.com").Return(u, nil)

	pair, err := uc.Execute(context.Background(), dto.LoginRequest{
		Email:    "alice@example.com",
		Password: "wrongpassword",
	})

	assert.Nil(t, pair)
	assert.ErrorIs(t, err, authErrors.ErrInvalidCredentials)
}
