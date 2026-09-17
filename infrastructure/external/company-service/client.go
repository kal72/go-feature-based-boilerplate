package companyservice

//go:generate mockgen -source=client.go -destination=mock/client_mock.go -package=mock

import (
	"context"
	"fmt"
	"time"

	companyservicepb "go-feature-based-boilerplate/gen/externalpb/company-service"
	"go-feature-based-boilerplate/infrastructure/config"

	"google.golang.org/grpc"
)

// Company represents the external company data structure.
type Company struct {
	ID      string
	Name    string
	Address string
	Status  string
}

// Client defines the contract for communicating with the external Company Service.
// Any consumers (e.g. usecase layer) depend on this interface, enabling easy mocking in tests.
type Client interface {
	GetCompanyByID(ctx context.Context, id string) (*Company, error)
}

// client is the concrete implementation of Client wrapping the gRPC stub.
type client struct {
	grpcClient companyservicepb.CompanyServiceClient
	timeout    time.Duration
}

// NewClient creates a new Client instance backed by a gRPC client connection.
func NewClient(conn *grpc.ClientConn, cfg *config.Config) Client {
	return NewClientWithStub(companyservicepb.NewCompanyServiceClient(conn), cfg)
}

// NewClientWithStub creates a Client instance using an injected CompanyServiceClient stub.
// This allows testing the adapter itself with a mocked protobuf gRPC stub without network overhead.
func NewClientWithStub(stub companyservicepb.CompanyServiceClient, cfg *config.Config) Client {
	timeout := cfg.CompanyClient.Timeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}

	return &client{
		grpcClient: stub,
		timeout:    timeout,
	}
}

// GetCompanyByID fetches company details by ID from the external Company Service.
func (c *client) GetCompanyByID(ctx context.Context, id string) (*Company, error) {
	ctxTimeout, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := c.grpcClient.GetCompany(ctxTimeout, &companyservicepb.GetCompanyRequest{
		Id: id,
	})
	if err != nil {
		return nil, fmt.Errorf("company-service client: get company failed for id %q: %w", id, err)
	}

	return &Company{
		ID:      resp.GetId(),
		Name:    resp.GetName(),
		Address: resp.GetAddress(),
		Status:  resp.GetStatus(),
	}, nil
}
