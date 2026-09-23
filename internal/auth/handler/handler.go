package handler

import (
	"context"

	pb "go-feature-based-boilerplate/gen/pb/auth"
	"go-feature-based-boilerplate/internal/auth/dto"
	"go-feature-based-boilerplate/internal/auth/usecase"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
)

// Handler implements the gRPC AuthService server interface.
type Handler struct {
	pb.UnimplementedAuthServiceServer
	loginUsecase   *usecase.LoginUsecase
	refreshUsecase *usecase.RefreshUsecase
	logoutUsecase  *usecase.LogoutUsecase
}

// NewHandler constructs a Handler with all required use cases.
func NewHandler(
	login *usecase.LoginUsecase,
	refresh *usecase.RefreshUsecase,
	logout *usecase.LogoutUsecase,
) *Handler {
	return &Handler{
		loginUsecase:   login,
		refreshUsecase: refresh,
		logoutUsecase:  logout,
	}
}

// Login authenticates a user and returns a token pair.
func (h *Handler) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	pair, err := h.loginUsecase.Execute(ctx, dto.LoginRequest{
		Email:    req.GetEmail(),
		Password: req.GetPassword(),
	})
	if err != nil {
		return nil, err
	}
	return &pb.LoginResponse{
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
	}, nil
}

// Refresh exchanges a refresh token for a new token pair.
func (h *Handler) Refresh(ctx context.Context, req *pb.RefreshRequest) (*pb.RefreshResponse, error) {
	pair, err := h.refreshUsecase.Execute(ctx, dto.RefreshTokenRequest{
		RefreshToken: req.GetRefreshToken(),
	})
	if err != nil {
		return nil, err
	}
	return &pb.RefreshResponse{
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
	}, nil
}

// Logout revokes the provided refresh token.
func (h *Handler) Logout(ctx context.Context, req *pb.LogoutRequest) (*pb.LogoutResponse, error) {
	if err := h.logoutUsecase.Execute(ctx, dto.LogoutRequest{
		RefreshToken: req.GetRefreshToken(),
	}); err != nil {
		return nil, err
	}
	return &pb.LogoutResponse{}, nil
}

// RegisterGRPC registers the auth service onto the provided gRPC server.
func (h *Handler) RegisterGRPC(server *grpc.Server) {
	pb.RegisterAuthServiceServer(server, h)
}

// RegisterGateway registers the auth service HTTP endpoints onto the gRPC-Gateway mux.
func (h *Handler) RegisterGateway(ctx context.Context, mux *runtime.ServeMux, grpcAddr string, opts []grpc.DialOption) error {
	return pb.RegisterAuthServiceHandlerFromEndpoint(ctx, mux, grpcAddr, opts)
}
