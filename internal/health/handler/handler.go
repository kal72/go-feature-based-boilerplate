package handler

import (
	"context"

	pb "go-feature-based-boilerplate/gen/pb/health"
	"go-feature-based-boilerplate/internal/health/checker"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Handler implements the gRPC HealthService server interface.
type Handler struct {
	pb.UnimplementedHealthServiceServer
	pinger checker.Pinger
}

// NewHandler constructs a Handler with a readiness pinger.
func NewHandler(pinger checker.Pinger) *Handler {
	return &Handler{pinger: pinger}
}

// Check is the liveness probe — returns SERVING as long as the process is up.
func (h *Handler) Check(_ context.Context, _ *pb.HealthCheckRequest) (*pb.HealthCheckResponse, error) {
	return &pb.HealthCheckResponse{
		Status:  pb.HealthCheckResponse_STATUS_SERVING,
		Message: "ok",
	}, nil
}

// Ready is the readiness probe — verifies dependencies are reachable.
func (h *Handler) Ready(ctx context.Context, _ *pb.HealthCheckRequest) (*pb.HealthCheckResponse, error) {
	if h.pinger != nil {
		if err := h.pinger.Ping(ctx); err != nil {
			return &pb.HealthCheckResponse{
				Status:  pb.HealthCheckResponse_STATUS_NOT_SERVING,
				Message: "dependency unavailable",
			}, status.Error(codes.Unavailable, "dependency unavailable")
		}
	}

	return &pb.HealthCheckResponse{
		Status:  pb.HealthCheckResponse_STATUS_SERVING,
		Message: "ok",
	}, nil
}
