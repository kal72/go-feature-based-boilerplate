package middleware_test

import (
	"context"
	"strconv"
	"testing"
	"time"

	"go-feature-based-boilerplate/infrastructure/config"
	"go-feature-based-boilerplate/infrastructure/middleware"
	"go-feature-based-boilerplate/pkg/contextutil"
	"go-feature-based-boilerplate/pkg/jwtutil"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestAuthInterceptor_PublicMethod(t *testing.T) {
	cfg := &config.Config{}
	cfg.JWT.SecretKey = "test-secret"
	interceptor := middleware.AuthInterceptor(cfg)

	called := false
	handler := func(ctx context.Context, req any) (any, error) {
		called = true
		return "ok", nil
	}

	info := &grpc.UnaryServerInfo{FullMethod: "/auth.v1.AuthService/Login"}
	resp, err := interceptor(context.Background(), nil, info, handler)

	assert.NoError(t, err)
	assert.Equal(t, "ok", resp)
	assert.True(t, called)
}

func TestAuthInterceptor_LogoutPublicMethod(t *testing.T) {
	cfg := &config.Config{}
	cfg.JWT.SecretKey = "test-secret"
	interceptor := middleware.AuthInterceptor(cfg)

	called := false
	handler := func(ctx context.Context, req any) (any, error) {
		called = true
		return "ok", nil
	}

	info := &grpc.UnaryServerInfo{FullMethod: "/auth.v1.AuthService/Logout"}
	resp, err := interceptor(context.Background(), nil, info, handler)

	assert.NoError(t, err)
	assert.Equal(t, "ok", resp)
	assert.True(t, called)
}

func TestAuthInterceptor_MissingMetadata(t *testing.T) {
	cfg := &config.Config{}
	cfg.JWT.SecretKey = "test-secret"
	interceptor := middleware.AuthInterceptor(cfg)

	info := &grpc.UnaryServerInfo{FullMethod: "/user.v1.UserService/GetUser"}
	_, err := interceptor(context.Background(), nil, info, func(ctx context.Context, req any) (any, error) {
		return nil, nil
	})

	assert.Error(t, err)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.Unauthenticated, st.Code())
}

func TestAuthInterceptor_InvalidHeaderFormat(t *testing.T) {
	cfg := &config.Config{}
	cfg.JWT.SecretKey = "test-secret"
	interceptor := middleware.AuthInterceptor(cfg)

	md := metadata.Pairs("authorization", "Basic abcdef")
	ctx := metadata.NewIncomingContext(context.Background(), md)

	info := &grpc.UnaryServerInfo{FullMethod: "/user.v1.UserService/GetUser"}
	_, err := interceptor(ctx, nil, info, func(ctx context.Context, req any) (any, error) {
		return nil, nil
	})

	assert.Error(t, err)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.Unauthenticated, st.Code())
}

func TestAuthInterceptor_ValidToken(t *testing.T) {
	secret := "test-secret"
	cfg := &config.Config{}
	cfg.JWT.SecretKey = secret
	interceptor := middleware.AuthInterceptor(cfg)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Subject:   strconv.FormatUint(42, 10),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	})
	tokenStr, err := token.SignedString([]byte(secret))
	assert.NoError(t, err)

	md := metadata.Pairs("authorization", "Bearer "+tokenStr)
	ctx := metadata.NewIncomingContext(context.Background(), md)

	var extractedUserID uint
	var found bool
	handler := func(ctx context.Context, req any) (any, error) {
		extractedUserID, found = contextutil.GetUserID(ctx)
		return "ok", nil
	}

	info := &grpc.UnaryServerInfo{FullMethod: "/user.v1.UserService/GetUser"}
	resp, err := interceptor(ctx, nil, info, handler)

	assert.NoError(t, err)
	assert.Equal(t, "ok", resp)
	assert.True(t, found)
	assert.Equal(t, uint(42), extractedUserID)
}

func TestAuthInterceptor_ZeroSubject_Rejected(t *testing.T) {
	secret := "test-secret"
	cfg := &config.Config{}
	cfg.JWT.SecretKey = secret
	interceptor := middleware.AuthInterceptor(cfg)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Subject:   "0",
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	})
	tokenStr, err := token.SignedString([]byte(secret))
	assert.NoError(t, err)

	md := metadata.Pairs("authorization", "Bearer "+tokenStr)
	ctx := metadata.NewIncomingContext(context.Background(), md)

	info := &grpc.UnaryServerInfo{FullMethod: "/user.v1.UserService/GetUser"}
	_, err = interceptor(ctx, nil, info, func(ctx context.Context, req any) (any, error) {
		return "ok", nil
	})

	assert.Error(t, err)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.Unauthenticated, st.Code())
}

func TestAuthInterceptor_ValidTokenWithRole(t *testing.T) {
	secret := "test-secret"
	cfg := &config.Config{}
	cfg.JWT.SecretKey = secret
	interceptor := middleware.AuthInterceptor(cfg)

	tokenStr, err := jwtutil.GenerateToken(42, "admin", []byte(secret), time.Hour)
	assert.NoError(t, err)

	md := metadata.Pairs("authorization", "Bearer "+tokenStr)
	ctx := metadata.NewIncomingContext(context.Background(), md)

	var extractedUserID uint
	var extractedRole string
	handler := func(ctx context.Context, req any) (any, error) {
		extractedUserID, _ = middleware.UserIDFromContext(ctx)
		extractedRole, _ = middleware.RoleFromContext(ctx)
		return "ok", nil
	}

	info := &grpc.UnaryServerInfo{FullMethod: "/user.v1.UserService/GetUser"}
	resp, err := interceptor(ctx, nil, info, handler)

	assert.NoError(t, err)
	assert.Equal(t, "ok", resp)
	assert.Equal(t, uint(42), extractedUserID)
	assert.Equal(t, "admin", extractedRole)
}
