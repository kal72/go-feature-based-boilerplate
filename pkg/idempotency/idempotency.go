// Package idempotency provides primitives and storage engines for ensuring idempotent
// operation execution across distributed microservices. It safeguards critical mutation
// operations (such as payments, orders, and fund transfers) from double processing caused
// by network retries or concurrent client requests.
package idempotency

import (
	"context"
	"errors"
	"strings"
	"time"

	"google.golang.org/grpc/metadata"
)

// Status represents the lifecycle phase of an idempotent operation.
type Status string

const (
	// StatusInProgress indicates an operation is currently executing.
	StatusInProgress Status = "IN_PROGRESS"

	// StatusCompleted indicates an operation finished successfully and its response is cached.
	StatusCompleted Status = "COMPLETED"
)

var (
	// ErrConcurrentRequest is returned when a second request arrives with the same key
	// while the first request is still actively executing (StatusInProgress).
	ErrConcurrentRequest = errors.New("idempotency: concurrent request in progress with the same idempotency key")
)

// Record holds the serialized execution state and cached response payload.
type Record struct {
	// Status tracks whether the operation is in progress or completed.
	Status Status `json:"status"`

	// Response contains the JSON-serialized return value of the operation.
	Response []byte `json:"response,omitempty"`

	// CreatedAt records when the operation was initially started.
	CreatedAt time.Time `json:"created_at"`
}

// Storage defines the persistence contract required for idempotency tracking.
// Implementations include in-memory storage (for local development/testing)
// and Redis (for distributed production environments).
type Storage interface {
	// Lock attempts to acquire an idempotency lock for key with a given TTL.
	// Returns true if the lock was successfully acquired (key was new).
	// If the key already exists, returns false.
	Lock(ctx context.Context, key string, ttl time.Duration) (bool, error)

	// Set stores the completed record for key with a given TTL.
	Set(ctx context.Context, key string, record *Record, ttl time.Duration) error

	// Get retrieves the recorded state for key.
	// Returns nil, nil if the key does not exist.
	Get(ctx context.Context, key string) (*Record, error)

	// Delete removes the key from storage (e.g., if execution failed and should allow retries).
	Delete(ctx context.Context, key string) error
}

// FromContext extracts the idempotency key from gRPC incoming metadata if present.
// It checks both "idempotency-key" and "x-idempotency-key" headers.
// Returns an empty string if neither header is provided.
func FromContext(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}

	for _, headerName := range []string{"idempotency-key", "x-idempotency-key"} {
		if values := md.Get(headerName); len(values) > 0 {
			trimmed := strings.TrimSpace(values[0])
			if trimmed != "" {
				return trimmed
			}
		}
	}

	return ""
}
