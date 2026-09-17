package usecase

import (
	"context"
	"time"

	"go-feature-based-boilerplate/internal/auth/dto"
	"go-feature-based-boilerplate/internal/auth/entity"
	authErrors "go-feature-based-boilerplate/internal/auth/errors"
	"go-feature-based-boilerplate/internal/auth/repository"
	"go-feature-based-boilerplate/internal/auth/validator"
	"go-feature-based-boilerplate/pkg/cryptoutil"
	"go-feature-based-boilerplate/pkg/jwtutil"
)

// LoginUsecase authenticates a user and issues a token pair.
type LoginUsecase struct {
	authRepo     repository.Repository
	userProvider UserProvider
	validator    *validator.Validator
	jwtSecret    []byte
	accessTTL    time.Duration
	refreshTTL   time.Duration
}

// NewLoginUsecase constructs a LoginUsecase.
func NewLoginUsecase(
	authRepo repository.Repository,
	userProvider UserProvider,
	v *validator.Validator,
	cfg TokenConfig,
) *LoginUsecase {
	return &LoginUsecase{
		authRepo:     authRepo,
		userProvider: userProvider,
		validator:    v,
		jwtSecret:    []byte(cfg.SecretKey),
		accessTTL:    cfg.AccessTTL,
		refreshTTL:   cfg.RefreshTTL,
	}
}

// Execute validates credentials, issues JWT access token and persists refresh token.
func (u *LoginUsecase) Execute(ctx context.Context, req dto.LoginRequest) (*dto.TokenPair, error) {
	if err := u.validator.Validate(req); err != nil {
		return nil, err
	}

	user, err := u.userProvider.FindByEmail(ctx, req.Email)
	if err != nil {
		// Map not-found to invalid credentials to avoid user enumeration.
		return nil, authErrors.ErrInvalidCredentials
	}

	if err := cryptoutil.CheckPassword(req.Password, user.Password); err != nil {
		return nil, authErrors.ErrInvalidCredentials
	}

	accessToken, err := jwtutil.GenerateToken(user.ID, string(user.Role), u.jwtSecret, u.accessTTL)
	if err != nil {
		return nil, err
	}

	rawRefresh, err := cryptoutil.GenerateRandomHex(32)
	if err != nil {
		return nil, err
	}

	refreshToken := &entity.RefreshToken{
		UserID:    user.ID,
		Token:     cryptoutil.HashSHA256(rawRefresh),
		ExpiresAt: time.Now().UTC().Add(u.refreshTTL),
	}
	if err := u.authRepo.CreateToken(ctx, refreshToken); err != nil {
		return nil, err
	}

	return &dto.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: rawRefresh,
	}, nil
}
