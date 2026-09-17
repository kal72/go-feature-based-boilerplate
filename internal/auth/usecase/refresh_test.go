package usecase_test

import (
	"context"
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

func TestRefreshUsecase_Success(t *testing.T) {
	authRepo := new(mockAuthRepository)
	userProv := new(mockUserProvider)
	v := validator.New()
	uc := usecase.NewRefreshUsecase(authRepo, userProv, v, testTokenConfig())

	rawToken := "valid-refresh-token-value-12345"
	tokenHash := cryptoutil.HashSHA256(rawToken)

	stored := &authEntity.RefreshToken{
		ID:        1,
		UserID:    42,
		Token:     tokenHash,
		Revoked:   false,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	u := &userEntity.User{
		ID:   42,
		Role: userEntity.RoleUser,
	}

	authRepo.On("FindTokenByValue", mock.Anything, tokenHash).Return(stored, nil)
	authRepo.On("RevokeToken", mock.Anything, tokenHash).Return(nil)
	userProv.On("FindByID", mock.Anything, uint(42)).Return(u, nil)
	authRepo.On("CreateToken", mock.Anything, mock.MatchedBy(func(t *authEntity.RefreshToken) bool {
		return t.UserID == 42
	})).Return(nil)

	pair, err := uc.Execute(context.Background(), dto.RefreshTokenRequest{
		RefreshToken: rawToken,
	})

	assert.NoError(t, err)
	assert.NotNil(t, pair)
	assert.NotEmpty(t, pair.AccessToken)
	assert.NotEmpty(t, pair.RefreshToken)
	assert.NotEqual(t, rawToken, pair.RefreshToken)
	authRepo.AssertExpectations(t)
	userProv.AssertExpectations(t)
}

func TestRefreshUsecase_CompromiseDetection(t *testing.T) {
	authRepo := new(mockAuthRepository)
	userProv := new(mockUserProvider)
	v := validator.New()
	uc := usecase.NewRefreshUsecase(authRepo, userProv, v, testTokenConfig())

	rawToken := "already-revoked-stolen-token"
	tokenHash := cryptoutil.HashSHA256(rawToken)

	stored := &authEntity.RefreshToken{
		ID:        2,
		UserID:    99,
		Token:     tokenHash,
		Revoked:   true, // Replay attack attempt!
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	authRepo.On("FindTokenByValue", mock.Anything, tokenHash).Return(stored, nil)
	// Must revoke all user tokens (family invalidation RFC 6819)
	authRepo.On("RevokeAllUserTokens", mock.Anything, uint(99)).Return(nil)

	pair, err := uc.Execute(context.Background(), dto.RefreshTokenRequest{
		RefreshToken: rawToken,
	})

	assert.Nil(t, pair)
	assert.ErrorIs(t, err, authErrors.ErrTokenRevoked)
	authRepo.AssertExpectations(t)
}

func TestRefreshUsecase_TokenExpired(t *testing.T) {
	authRepo := new(mockAuthRepository)
	userProv := new(mockUserProvider)
	v := validator.New()
	uc := usecase.NewRefreshUsecase(authRepo, userProv, v, testTokenConfig())

	rawToken := "expired-token-12345"
	tokenHash := cryptoutil.HashSHA256(rawToken)

	stored := &authEntity.RefreshToken{
		ID:        3,
		UserID:    10,
		Token:     tokenHash,
		Revoked:   false,
		ExpiresAt: time.Now().Add(-1 * time.Hour), // Expired
	}

	authRepo.On("FindTokenByValue", mock.Anything, tokenHash).Return(stored, nil)

	pair, err := uc.Execute(context.Background(), dto.RefreshTokenRequest{
		RefreshToken: rawToken,
	})

	assert.Nil(t, pair)
	assert.ErrorIs(t, err, authErrors.ErrTokenExpired)
	authRepo.AssertExpectations(t)
}

func TestRefreshUsecase_TokenNotFound(t *testing.T) {
	authRepo := new(mockAuthRepository)
	userProv := new(mockUserProvider)
	v := validator.New()
	uc := usecase.NewRefreshUsecase(authRepo, userProv, v, testTokenConfig())

	rawToken := "unknown-token"
	tokenHash := cryptoutil.HashSHA256(rawToken)

	authRepo.On("FindTokenByValue", mock.Anything, tokenHash).Return(nil, authErrors.ErrTokenNotFound)

	pair, err := uc.Execute(context.Background(), dto.RefreshTokenRequest{
		RefreshToken: rawToken,
	})

	assert.Nil(t, pair)
	assert.ErrorIs(t, err, authErrors.ErrTokenNotFound)
	authRepo.AssertExpectations(t)
}
