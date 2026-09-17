package contextutil

import "context"

type contextKey string

const (
	// ContextKeyUserID is the context key for storing the authenticated user ID.
	ContextKeyUserID contextKey = "user_id"
	// ContextKeyRole is the context key for storing the authenticated user role.
	ContextKeyRole contextKey = "role"
	// ContextKeyRequestID is the context key for storing the unique request ID.
	ContextKeyRequestID contextKey = "request_id"
)

// WithRequestID injects the unique request ID into the context.
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, ContextKeyRequestID, requestID)
}

// GetRequestID retrieves the unique request ID from context.
func GetRequestID(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(ContextKeyRequestID).(string)
	return v, ok
}

// WithUserID injects the authenticated user ID into the context.
func WithUserID(ctx context.Context, userID uint) context.Context {
	return context.WithValue(ctx, ContextKeyUserID, userID)
}

// GetUserID retrieves the authenticated user ID from context.
func GetUserID(ctx context.Context) (uint, bool) {
	v, ok := ctx.Value(ContextKeyUserID).(uint)
	return v, ok
}

// WithRole injects the authenticated user role into the context.
func WithRole(ctx context.Context, role string) context.Context {
	return context.WithValue(ctx, ContextKeyRole, role)
}

// GetRole retrieves the authenticated user role from context.
func GetRole(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(ContextKeyRole).(string)
	return v, ok
}
