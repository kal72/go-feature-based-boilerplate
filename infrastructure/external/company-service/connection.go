package companyservice

import (
	"fmt"

	"go-feature-based-boilerplate/infrastructure/config"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// NewConnection establishes a managed gRPC client connection to the external Company Service.
// It instruments the connection with OpenTelemetry stats handler for automatic distributed trace propagation.
func NewConnection(cfg *config.Config) (*grpc.ClientConn, func(), error) {
	conn, err := grpc.NewClient(
		cfg.CompanyClient.GRPCAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("company-service: dial %s: %w", cfg.CompanyClient.GRPCAddr, err)
	}

	cleanup := func() {
		_ = conn.Close()
	}

	return conn, cleanup, nil
}
