package middleware_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"go-feature-based-boilerplate/infrastructure/middleware"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestTimeoutInterceptor_CompletesWithinTimeout(t *testing.T) {
	interceptor := middleware.TimeoutInterceptor(500 * time.Millisecond)
	info := &grpc.UnaryServerInfo{FullMethod: "/user.v1.UserService/GetUser"}

	resp, err := interceptor(context.Background(), "req", info, func(ctx context.Context, req any) (any, error) {
		// Verify context has a deadline
		deadline, ok := ctx.Deadline()
		assert.True(t, ok)
		assert.True(t, time.Until(deadline) > 0)
		return "success", nil
	})

	require.NoError(t, err)
	assert.Equal(t, "success", resp)
}

func TestTimeoutInterceptor_ExceedsTimeout(t *testing.T) {
	// Short timeout for fast testing
	interceptor := middleware.TimeoutInterceptor(50 * time.Millisecond)
	info := &grpc.UnaryServerInfo{FullMethod: "/user.v1.UserService/GetUser"}

	resp, err := interceptor(context.Background(), "req", info, func(ctx context.Context, req any) (any, error) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(200 * time.Millisecond):
			return "too-late", nil
		}
	})

	require.Error(t, err)
	assert.Nil(t, resp)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.DeadlineExceeded, st.Code())
	assert.Equal(t, "deadline exceeded", st.Message())
}

func TestTimeoutInterceptor_ClientTighterDeadlinePreserved(t *testing.T) {
	interceptor := middleware.TimeoutInterceptor(10 * time.Second)
	info := &grpc.UnaryServerInfo{FullMethod: "/user.v1.UserService/GetUser"}

	// Client provides a tight deadline of 50ms
	clientCtx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	var observedRemaining time.Duration
	_, err := interceptor(clientCtx, "req", info, func(ctx context.Context, req any) (any, error) {
		deadline, ok := ctx.Deadline()
		assert.True(t, ok)
		observedRemaining = time.Until(deadline)
		return "ok", nil
	})

	require.NoError(t, err)
	// Must be <= 50ms (not extended to 10s)
	assert.True(t, observedRemaining <= 50*time.Millisecond)
}

func TestTimeoutInterceptor_ClientLooserDeadlineTruncated(t *testing.T) {
	serverTimeout := 100 * time.Millisecond
	interceptor := middleware.TimeoutInterceptor(serverTimeout)
	info := &grpc.UnaryServerInfo{FullMethod: "/user.v1.UserService/GetUser"}

	// Client provides a loose deadline of 10 minutes
	clientCtx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	var observedRemaining time.Duration
	_, err := interceptor(clientCtx, "req", info, func(ctx context.Context, req any) (any, error) {
		deadline, ok := ctx.Deadline()
		assert.True(t, ok)
		observedRemaining = time.Until(deadline)
		return "ok", nil
	})

	require.NoError(t, err)
	// Server must truncate to <= 100ms
	assert.True(t, observedRemaining <= 100*time.Millisecond)
}

func TestTimeoutInterceptor_MethodSpecificOverride(t *testing.T) {
	overrides := map[string]time.Duration{
		"/user.v1.UserService/ExportData": 300 * time.Millisecond,
	}
	defaultTimeout := 50 * time.Millisecond
	interceptor := middleware.TimeoutInterceptor(defaultTimeout, overrides)

	// 1. Regular method should use default timeout (50ms)
	regularInfo := &grpc.UnaryServerInfo{FullMethod: "/user.v1.UserService/GetUser"}
	var regularRemaining time.Duration
	_, err := interceptor(context.Background(), "req", regularInfo, func(ctx context.Context, req any) (any, error) {
		deadline, ok := ctx.Deadline()
		assert.True(t, ok)
		regularRemaining = time.Until(deadline)
		return "ok", nil
	})
	require.NoError(t, err)
	assert.True(t, regularRemaining <= 50*time.Millisecond)

	// 2. Overridden method should use custom timeout (300ms)
	exportInfo := &grpc.UnaryServerInfo{FullMethod: "/user.v1.UserService/ExportData"}
	var exportRemaining time.Duration
	_, err = interceptor(context.Background(), "req", exportInfo, func(ctx context.Context, req any) (any, error) {
		deadline, ok := ctx.Deadline()
		assert.True(t, ok)
		exportRemaining = time.Until(deadline)
		return "ok", nil
	})
	require.NoError(t, err)
	assert.True(t, exportRemaining > 100*time.Millisecond)
	assert.True(t, exportRemaining <= 300*time.Millisecond)
}

func TestTimeoutInterceptor_ClientCancellation(t *testing.T) {
	interceptor := middleware.TimeoutInterceptor(5 * time.Second)
	info := &grpc.UnaryServerInfo{FullMethod: "/user.v1.UserService/GetUser"}

	clientCtx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := interceptor(clientCtx, "req", info, func(ctx context.Context, req any) (any, error) {
		return nil, ctx.Err()
	})

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Canceled, st.Code())
}

func TestTimeoutInterceptor_HandlerReturnsContextDeadlineExceededDirectly(t *testing.T) {
	interceptor := middleware.TimeoutInterceptor(1 * time.Second)
	info := &grpc.UnaryServerInfo{FullMethod: "/user.v1.UserService/GetUser"}

	_, err := interceptor(context.Background(), "req", info, func(ctx context.Context, req any) (any, error) {
		return nil, errors.New("something wrapped: " + context.DeadlineExceeded.Error())
	})

	// Non-context error should pass through as-is
	require.Error(t, err)
}
