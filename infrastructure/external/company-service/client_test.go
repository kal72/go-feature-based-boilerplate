package companyservice_test

import (
	"context"
	"errors"
	"testing"
	"time"

	companyservicepb "go-feature-based-boilerplate/gen/externalpb/company-service"
	"go-feature-based-boilerplate/infrastructure/config"
	companyservice "go-feature-based-boilerplate/infrastructure/external/company-service"
	companymock "go-feature-based-boilerplate/infrastructure/external/company-service/mock"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc"
)

// mockStub implements companyservicepb.CompanyServiceClient for unit testing without network.
type mockStub struct {
	getCompanyFn func(ctx context.Context, in *companyservicepb.GetCompanyRequest, opts ...grpc.CallOption) (*companyservicepb.GetCompanyResponse, error)
}

func (m *mockStub) GetCompany(ctx context.Context, in *companyservicepb.GetCompanyRequest, opts ...grpc.CallOption) (*companyservicepb.GetCompanyResponse, error) {
	return m.getCompanyFn(ctx, in, opts...)
}

func TestClient_GetCompanyByID_Success(t *testing.T) {
	stub := &mockStub{
		getCompanyFn: func(_ context.Context, req *companyservicepb.GetCompanyRequest, _ ...grpc.CallOption) (*companyservicepb.GetCompanyResponse, error) {
			return &companyservicepb.GetCompanyResponse{
				Id:      req.GetId(),
				Name:    "Acme Corp",
				Address: "123 Innovation Drive",
				Status:  "ACTIVE",
			}, nil
		},
	}

	cfg := &config.Config{
		CompanyClient: config.CompanyClientConfig{
			Timeout: 2 * time.Second,
		},
	}

	client := companyservice.NewClientWithStub(stub, cfg)
	comp, err := client.GetCompanyByID(context.Background(), "comp-123")

	require.NoError(t, err)
	require.NotNil(t, comp)
	assert.Equal(t, "comp-123", comp.ID)
	assert.Equal(t, "Acme Corp", comp.Name)
	assert.Equal(t, "123 Innovation Drive", comp.Address)
	assert.Equal(t, "ACTIVE", comp.Status)
}

func TestClient_GetCompanyByID_Error(t *testing.T) {
	stub := &mockStub{
		getCompanyFn: func(_ context.Context, _ *companyservicepb.GetCompanyRequest, _ ...grpc.CallOption) (*companyservicepb.GetCompanyResponse, error) {
			return nil, errors.New("rpc error: status = NotFound desc = company not found")
		},
	}

	cfg := &config.Config{
		CompanyClient: config.CompanyClientConfig{
			Timeout: 2 * time.Second,
		},
	}

	client := companyservice.NewClientWithStub(stub, cfg)
	comp, err := client.GetCompanyByID(context.Background(), "comp-404")

	require.Error(t, err)
	assert.Nil(t, comp)
	assert.Contains(t, err.Error(), "company not found")
}

func TestMockClient_Demonstration(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := companymock.NewMockClient(ctrl)
	mockClient.EXPECT().
		GetCompanyByID(gomock.Any(), "comp-777").
		Return(&companyservice.Company{
			ID:   "comp-777",
			Name: "Mocked Enterprise",
		}, nil)

	comp, err := mockClient.GetCompanyByID(context.Background(), "comp-777")
	require.NoError(t, err)
	assert.Equal(t, "Mocked Enterprise", comp.Name)
}
