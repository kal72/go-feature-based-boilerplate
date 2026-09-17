package auth

import (
	"go-feature-based-boilerplate/internal/auth/handler"
	"go-feature-based-boilerplate/internal/auth/repository"
	"go-feature-based-boilerplate/internal/auth/repository/postgres"
	"go-feature-based-boilerplate/internal/auth/usecase"
	"go-feature-based-boilerplate/internal/auth/validator"

	userRepo "go-feature-based-boilerplate/internal/user/repository"

	"github.com/google/wire"
)

// ProvideUserProvider adapts userRepo.Repository to usecase.UserProvider.
func ProvideUserProvider(repo userRepo.Repository) usecase.UserProvider {
	return repo
}

// ProviderSet wires all auth feature components.
var ProviderSet = wire.NewSet(
	validator.New,
	usecase.NewTokenConfig,
	usecase.NewLoginUsecase,
	usecase.NewRefreshUsecase,
	usecase.NewLogoutUsecase,
	handler.NewHandler,
	postgres.NewAuthRepository,
	ProvideUserProvider,
	wire.Bind(new(repository.Repository), new(*postgres.AuthRepository)),
)
