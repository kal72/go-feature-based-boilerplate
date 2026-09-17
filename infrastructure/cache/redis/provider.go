package redis

import "github.com/google/wire"

// ProviderSet wires the Redis client.
var ProviderSet = wire.NewSet(New)
