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

// RefreshUsecase exchanges a valid refresh token for a new token pair.
type RefreshUsecase struct {
	authRepo     repository.Repository
	userProvider UserProvider
	validator    *validator.Validator
	jwtSecret    []byte
	accessTTL    time.Duration
	refreshTTL   time.Duration
}

// NewRefreshUsecase constructs a RefreshUsecase.
func NewRefreshUsecase(
	authRepo repository.Repository,
	userProvider UserProvider,
	v *validator.Validator,
	cfg TokenConfig,
) *RefreshUsecase {
	return &RefreshUsecase{
		authRepo:     authRepo,
		userProvider: userProvider,
		validator:    v,
		jwtSecret:    []byte(cfg.SecretKey),
		accessTTL:    cfg.AccessTTL,
		refreshTTL:   cfg.RefreshTTL,
	}
}

// Execute validates the refresh token and issues a new token pair (token rotation).
func (u *RefreshUsecase) Execute(ctx context.Context, req dto.RefreshTokenRequest) (*dto.TokenPair, error) {
	if err := u.validator.Validate(req); err != nil {
		return nil, err
	}

	tokenHash := cryptoutil.HashSHA256(req.RefreshToken)
	stored, err := u.authRepo.FindTokenByValue(ctx, tokenHash)
	if err != nil {
		return nil, authErrors.ErrTokenNotFound
	}

	if stored.Revoked {
		// RFC 6819 Section 5.2.2.3: Token reuse indicates possible compromise or replay attack.
		// Invalidate all tokens in the user's family to protect the account.
		_ = u.authRepo.RevokeAllUserTokens(ctx, stored.UserID)
		return nil, authErrors.ErrTokenRevoked
	}
	if stored.IsExpired() {
		return nil, authErrors.ErrTokenExpired
	}

	// Revoke old token (rotation).
	if err := u.authRepo.RevokeToken(ctx, tokenHash); err != nil {
		return nil, err
	}

	user, err := u.userProvider.FindByID(ctx, stored.UserID)
	if err != nil {
		return nil, err
	}

	accessToken, err := jwtutil.GenerateToken(user.ID, string(user.Role), u.jwtSecret, u.accessTTL)
	if err != nil {
		return nil, err
	}

	rawRefresh, err := cryptoutil.GenerateRandomHex(32)
	if err != nil {
		return nil, err
	}

	newToken := &entity.RefreshToken{
		UserID:    user.ID,
		Token:     cryptoutil.HashSHA256(rawRefresh),
		ExpiresAt: time.Now().UTC().Add(u.refreshTTL),
	}
	if err := u.authRepo.CreateToken(ctx, newToken); err != nil {
		return nil, err
	}

	return &dto.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: rawRefresh,
	}, nil
}
