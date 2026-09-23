package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go-feature-based-boilerplate/infrastructure/config"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestResponseEnvelopeMiddleware_Success(t *testing.T) {
	// Handler returning standard JSON
	innerHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":123,"username":"johndoe"}`))
	})

	wrapped := responseEnvelopeMiddleware(innerHandler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/123", nil)
	req.Header.Set("X-Request-ID", "custom-req-id-001")
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "custom-req-id-001", rec.Header().Get("X-Request-ID"))
	assert.Contains(t, rec.Header().Get("Content-Type"), "application/json")

	var env SuccessEnvelope
	err := json.Unmarshal(rec.Body.Bytes(), &env)
	require.NoError(t, err)

	assert.Equal(t, "success", env.Status)
	assert.Equal(t, 200, env.Code)
	assert.Equal(t, "success", env.Message)
	assert.Equal(t, "custom-req-id-001", env.Meta.RequestID)

	_, parseTimeErr := time.Parse(time.RFC3339, env.Meta.Timestamp)
	assert.NoError(t, parseTimeErr, "timestamp must be RFC3339 formatted")

	var data map[string]any
	err = json.Unmarshal(env.Data, &data)
	require.NoError(t, err)
	assert.Equal(t, float64(123), data["id"])
	assert.Equal(t, "johndoe", data["username"])
}

func TestResponseEnvelopeMiddleware_GeneratesRequestID(t *testing.T) {
	innerHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"SERVING"}`))
	})

	wrapped := responseEnvelopeMiddleware(innerHandler)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	genReqID := rec.Header().Get("X-Request-ID")
	assert.NotEmpty(t, genReqID)

	var env SuccessEnvelope
	err := json.Unmarshal(rec.Body.Bytes(), &env)
	require.NoError(t, err)

	assert.Equal(t, "success", env.Status)
	assert.Equal(t, genReqID, env.Meta.RequestID)
}

func TestGatewayErrorHandler_Formatting(t *testing.T) {
	tests := []struct {
		name           string
		grpcErr        error
		expectedStatus int
		expectedMsg    string
	}{
		{
			name:           "InvalidArgument maps to 400",
			grpcErr:        status.Error(codes.InvalidArgument, "invalid email address"),
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "invalid email address",
		},
		{
			name:           "NotFound maps to 404",
			grpcErr:        status.Error(codes.NotFound, "user not found"),
			expectedStatus: http.StatusNotFound,
			expectedMsg:    "user not found",
		},
		{
			name:           "Unauthenticated maps to 401",
			grpcErr:        status.Error(codes.Unauthenticated, "token expired"),
			expectedStatus: http.StatusUnauthorized,
			expectedMsg:    "token expired",
		},
		{
			name:           "PermissionDenied maps to 403",
			grpcErr:        status.Error(codes.PermissionDenied, "forbidden resource"),
			expectedStatus: http.StatusForbidden,
			expectedMsg:    "forbidden resource",
		},
		{
			name:           "Internal maps to 500",
			grpcErr:        status.Error(codes.Internal, "database error"),
			expectedStatus: http.StatusInternalServerError,
			expectedMsg:    "database error",
		},
		{
			name:           "Generic error defaults to 500",
			grpcErr:        errors.New("unexpected error"),
			expectedStatus: http.StatusInternalServerError,
			expectedMsg:    "unexpected error",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
			req.Header.Set("X-Request-ID", "err-req-123")
			rec := httptest.NewRecorder()

			GatewayErrorHandler(context.Background(), nil, nil, rec, req, tc.grpcErr)

			assert.Equal(t, tc.expectedStatus, rec.Code)
			assert.Contains(t, rec.Header().Get("Content-Type"), "application/json")

			var env ErrorEnvelope
			err := json.Unmarshal(rec.Body.Bytes(), &env)
			require.NoError(t, err)

			assert.Equal(t, "failed", env.Status)
			assert.Equal(t, tc.expectedStatus, env.Code)
			assert.Equal(t, tc.expectedMsg, env.Message)
			assert.Nil(t, env.Errors)
			assert.Equal(t, "err-req-123", env.Meta.RequestID)

			_, parseTimeErr := time.Parse(time.RFC3339, env.Meta.Timestamp)
			assert.NoError(t, parseTimeErr)
		})
	}
}

func TestResponseEnvelopeMiddleware_PassesThroughErrors(t *testing.T) {
	// Handler that simulates an error already written by GatewayErrorHandler
	innerHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		GatewayErrorHandler(r.Context(), nil, nil, w, r, status.Error(codes.NotFound, "item not found"))
	})

	wrapped := responseEnvelopeMiddleware(innerHandler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/items/99", nil)
	req.Header.Set("X-Request-ID", "req-err-passthrough")
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)

	var env ErrorEnvelope
	err := json.Unmarshal(rec.Body.Bytes(), &env)
	require.NoError(t, err)

	assert.Equal(t, "failed", env.Status)
	assert.Equal(t, 404, env.Code)
	assert.Equal(t, "item not found", env.Message)
	assert.Equal(t, "req-err-passthrough", env.Meta.RequestID)
}

func TestWithCORS(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	handler := withCORS(inner)

	// OPTIONS preflight
	reqOptions := httptest.NewRequest(http.MethodOptions, "/api/v1/auth/login", nil)
	recOptions := httptest.NewRecorder()
	handler.ServeHTTP(recOptions, reqOptions)

	assert.Equal(t, http.StatusNoContent, recOptions.Code)
	assert.Equal(t, "*", recOptions.Header().Get("Access-Control-Allow-Origin"))
	assert.Contains(t, recOptions.Header().Get("Access-Control-Allow-Methods"), "POST")

	// Standard GET
	reqGet := httptest.NewRequest(http.MethodGet, "/api/v1/users/1", nil)
	recGet := httptest.NewRecorder()
	handler.ServeHTTP(recGet, reqGet)

	assert.Equal(t, http.StatusOK, recGet.Code)
	assert.Equal(t, "*", recGet.Header().Get("Access-Control-Allow-Origin"))
}

func TestNewHTTPGateway_TelemetryAndPprof(t *testing.T) {
	cfg := &config.Config{
		App: config.AppConfig{
			Environment: "development",
		},
		Server: config.ServerConfig{
			HTTPPort:     8080,
			GRPCPort:     50051,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 5 * time.Second,
		},
		Telemetry: config.TelemetryConfig{
			Enabled: true,
		},
	}

	svcs := NewServices()
	srv, err := NewHTTPGateway(context.Background(), cfg, svcs)
	require.NoError(t, err)
	require.NotNil(t, srv)
	require.NotNil(t, srv.Handler)

	// Test pprof index endpoint in development
	req := httptest.NewRequest(http.MethodGet, "/debug/pprof/", nil)
	rec := httptest.NewRecorder()
	srv.Handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "Types of profiles available")

	// Test production environment does not register pprof
	cfgProd := &config.Config{
		App: config.AppConfig{
			Environment: "production",
		},
		Server: config.ServerConfig{
			HTTPPort:     8080,
			GRPCPort:     50051,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 5 * time.Second,
		},
		Telemetry: config.TelemetryConfig{
			Enabled: false,
		},
	}

	srvProd, err := NewHTTPGateway(context.Background(), cfgProd, svcs)
	require.NoError(t, err)
	require.NotNil(t, srvProd)

	recProd := httptest.NewRecorder()
	srvProd.Handler.ServeHTTP(recProd, req)
	// In production without pprof, /debug/pprof/ falls through to grpc-gateway which yields 404
	assert.Equal(t, http.StatusNotFound, recProd.Code)
}
