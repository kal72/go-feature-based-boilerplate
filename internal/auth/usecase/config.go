package usecase

import (
	"time"

	"go-feature-based-boilerplate/infrastructure/config"
)

// TokenConfig holds the JWT signing parameters needed by auth use cases.
type TokenConfig struct {
	SecretKey  string
	AccessTTL  time.Duration
	RefreshTTL time.Duration
}

// NewTokenConfig extracts JWT settings from the application config.
func NewTokenConfig(cfg *config.Config) TokenConfig {
	return TokenConfig{
		SecretKey:  cfg.JWT.SecretKey,
		AccessTTL:  cfg.JWT.AccessTokenTTL,
		RefreshTTL: cfg.JWT.RefreshTokenTTL,
	}
}
