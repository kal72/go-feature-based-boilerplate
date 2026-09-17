package transaction

import "context"

// Manager defines the interface for running atomic database transactions.
// It allows usecase layers to manage unit-of-work boundaries without
// importing or leaking infrastructure-specific database objects.
type Manager interface {
	// RunInTransaction executes the given function inside a database transaction.
	// If the function returns nil, the transaction is committed.
	// If the function returns an error or panics, the transaction is rolled back.
	RunInTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}
