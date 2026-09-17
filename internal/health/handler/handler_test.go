package handler_test

import (
	"context"
	"errors"
	"testing"

	pb "go-feature-based-boilerplate/gen/pb/health"
	"go-feature-based-boilerplate/internal/health/handler"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type mockPinger struct {
	err error
}

func (m *mockPinger) Ping(ctx context.Context) error {
	return m.err
}

func TestHandler_Check(t *testing.T) {
	h := handler.NewHandler(&mockPinger{})
	resp, err := h.Check(context.Background(), &pb.HealthCheckRequest{})
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, pb.HealthCheckResponse_STATUS_SERVING, resp.Status)
	assert.Equal(t, "ok", resp.Message)
}

func TestHandler_Ready_Success(t *testing.T) {
	h := handler.NewHandler(&mockPinger{err: nil})
	resp, err := h.Ready(context.Background(), &pb.HealthCheckRequest{})
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, pb.HealthCheckResponse_STATUS_SERVING, resp.Status)
	assert.Equal(t, "ok", resp.Message)
}

func TestHandler_Ready_Unavailable(t *testing.T) {
	h := handler.NewHandler(&mockPinger{err: errors.New("db connection lost")})
	resp, err := h.Ready(context.Background(), &pb.HealthCheckRequest{})
	assert.Error(t, err)

	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.Unavailable, st.Code())
	assert.Equal(t, "dependency unavailable", st.Message())

	assert.NotNil(t, resp)
	assert.Equal(t, pb.HealthCheckResponse_STATUS_NOT_SERVING, resp.Status)
}

func TestHandler_Ready_NilPinger(t *testing.T) {
	h := handler.NewHandler(nil)
	resp, err := h.Ready(context.Background(), &pb.HealthCheckRequest{})
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, pb.HealthCheckResponse_STATUS_SERVING, resp.Status)
}
