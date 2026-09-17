package health

import (
	"go-feature-based-boilerplate/internal/health/checker"
	"go-feature-based-boilerplate/internal/health/handler"

	"github.com/google/wire"
)

// ProviderSet wires all health feature components.
var ProviderSet = wire.NewSet(
	checker.NewDBPinger,
	wire.Bind(new(checker.Pinger), new(*checker.DBPinger)),
	handler.NewHandler,
)
