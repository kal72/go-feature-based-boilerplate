package companyservice

import "github.com/google/wire"

// ProviderSet exports the external company service connection and client dependencies for Wire DI.
var ProviderSet = wire.NewSet(
	NewConnection,
	NewClient,
)
