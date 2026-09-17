package middleware_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"go-feature-based-boilerplate/infrastructure/middleware"
	"go-feature-based-boilerplate/pkg/errorutil"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestErrorInterceptor_Success(t *testing.T) {
	interceptor := middleware.ErrorInterceptor()
	handler := func(ctx context.Context, req any) (any, error) {
		return "ok", nil
	}

	resp, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{}, handler)
	assert.NoError(t, err)
	assert.Equal(t, "ok", resp)
}

func TestErrorInterceptor_AppError_NotFound(t *testing.T) {
	interceptor := middleware.ErrorInterceptor()
	handler := func(ctx context.Context, req any) (any, error) {
		return nil, errorutil.New(errorutil.CodeNotFound, "user not found")
	}

	_, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{}, handler)
	assert.Error(t, err)

	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code())
	assert.Equal(t, "user not found", st.Message())
}

func TestErrorInterceptor_AppError_InvalidInput_WithCause(t *testing.T) {
	interceptor := middleware.ErrorInterceptor()
	cause := errors.New("email is required")
	handler := func(ctx context.Context, req any) (any, error) {
		return nil, errorutil.Wrap(errorutil.CodeInvalidInput, "validation failed", cause)
	}

	_, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{}, handler)
	assert.Error(t, err)

	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Equal(t, "validation failed: email is required", st.Message())
}

func TestErrorInterceptor_AppError_Internal_Sanitized(t *testing.T) {
	interceptor := middleware.ErrorInterceptor()
	cause := errors.New("database connection timeout: dial tcp 10.0.0.1:5432: i/o timeout")
	handler := func(ctx context.Context, req any) (any, error) {
		return nil, errorutil.Wrap(errorutil.CodeInternal, "failed to query database", cause)
	}

	_, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{}, handler)
	assert.Error(t, err)

	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())
	assert.Equal(t, "failed to query database", st.Message())
}

func TestErrorInterceptor_PreservesGRPCStatus(t *testing.T) {
	interceptor := middleware.ErrorInterceptor()
	handler := func(ctx context.Context, req any) (any, error) {
		return nil, status.Error(codes.Unavailable, "dependency unavailable")
	}

	_, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{}, handler)
	assert.Error(t, err)

	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.Unavailable, st.Code())
	assert.Equal(t, "dependency unavailable", st.Message())
}

func TestErrorInterceptor_UnknownError_MapsToInternal(t *testing.T) {
	interceptor := middleware.ErrorInterceptor()
	handler := func(ctx context.Context, req any) (any, error) {
		return nil, fmt.Errorf("unexpected panic or foreign error")
	}

	_, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{}, handler)
	assert.Error(t, err)

	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())
	assert.Equal(t, "internal server error", st.Message())
}
