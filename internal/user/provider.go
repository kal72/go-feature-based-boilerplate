package user

import (
	"go-feature-based-boilerplate/internal/user/handler"
	"go-feature-based-boilerplate/internal/user/repository"
	"go-feature-based-boilerplate/internal/user/repository/postgres"
	"go-feature-based-boilerplate/internal/user/usecase"
	"go-feature-based-boilerplate/internal/user/validator"

	"github.com/google/wire"
)

// ProviderSet wires all user feature components.
var ProviderSet = wire.NewSet(
	validator.New,
	usecase.NewCreateUsecase,
	usecase.NewFindUsecase,
	usecase.NewUpdateUsecase,
	usecase.NewDeleteUsecase,
	handler.NewHandler,
	postgres.NewUserRepository,
	wire.Bind(new(repository.Repository), new(*postgres.UserRepository)),
)
