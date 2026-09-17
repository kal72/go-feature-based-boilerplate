package mongo

import "github.com/google/wire"

// ProviderSet wires the MongoDB connection.
var ProviderSet = wire.NewSet(New)
