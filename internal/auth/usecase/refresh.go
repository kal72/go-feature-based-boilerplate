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
	"go-feature-based-boilerplate/pkg/transaction"
)

// RefreshUsecase exchanges a valid refresh token for a new token pair.
type RefreshUsecase struct {
	authRepo     repository.Repository
	userProvider UserProvider
	validator    *validator.Validator
	txManager    transaction.Manager
	jwtSecret    []byte
	accessTTL    time.Duration
	refreshTTL   time.Duration
}

// NewRefreshUsecase constructs a RefreshUsecase.
func NewRefreshUsecase(
	authRepo repository.Repository,
	userProvider UserProvider,
	v *validator.Validator,
	txManager transaction.Manager,
	cfg TokenConfig,
) *RefreshUsecase {
	return &RefreshUsecase{
		authRepo:     authRepo,
		userProvider: userProvider,
		validator:    v,
		txManager:    txManager,
		jwtSecret:    []byte(cfg.SecretKey),
		accessTTL:    cfg.AccessTTL,
		refreshTTL:   cfg.RefreshTTL,
	}
}

// Execute validates the refresh token and issues a new token pair (token rotation).
// All database operations are wrapped in an atomic transaction to ensure safe rotation.
func (u *RefreshUsecase) Execute(ctx context.Context, req dto.RefreshTokenRequest) (*dto.TokenPair, error) {
	if err := u.validator.Validate(req); err != nil {
		return nil, err
	}

	tokenHash := cryptoutil.HashSHA256(req.RefreshToken)

	var tokenPair *dto.TokenPair
	var isReplayAttack bool
	var replayUserID uint

	err := u.txManager.RunInTransaction(ctx, func(txCtx context.Context) error {
		stored, err := u.authRepo.FindTokenByValue(txCtx, tokenHash)
		if err != nil {
			return authErrors.ErrTokenNotFound
		}

		if stored.Revoked {
			// RFC 6819 Section 5.2.2.3: Token reuse indicates possible compromise or replay attack.
			isReplayAttack = true
			replayUserID = stored.UserID
			return authErrors.ErrTokenRevoked
		}
		if stored.IsExpired() {
			return authErrors.ErrTokenExpired
		}

		// Revoke old token (rotation).
		if err := u.authRepo.RevokeToken(txCtx, tokenHash); err != nil {
			return err
		}

		user, err := u.userProvider.FindByID(txCtx, stored.UserID)
		if err != nil {
			return err
		}

		accessToken, err := jwtutil.GenerateToken(user.ID, string(user.Role), u.jwtSecret, u.accessTTL)
		if err != nil {
			return err
		}

		rawRefresh, err := cryptoutil.GenerateRandomHex(32)
		if err != nil {
			return err
		}

		newToken := &entity.RefreshToken{
			UserID:    user.ID,
			Token:     cryptoutil.HashSHA256(rawRefresh),
			ExpiresAt: time.Now().UTC().Add(u.refreshTTL),
		}
		if err := u.authRepo.CreateToken(txCtx, newToken); err != nil {
			return err
		}

		tokenPair = &dto.TokenPair{
			AccessToken:  accessToken,
			RefreshToken: rawRefresh,
		}
		return nil
	})

	if isReplayAttack {
		// Invalidate all tokens in the user's family to protect the account against replay attack.
		// Executed outside the aborted transaction so all session revocations are committed.
		_ = u.authRepo.RevokeAllUserTokens(ctx, replayUserID)
		return nil, authErrors.ErrTokenRevoked
	}
	if err != nil {
		return nil, err
	}

	return tokenPair, nil
}
