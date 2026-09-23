package handler

import (
	"context"

	pb "go-feature-based-boilerplate/gen/pb/user"
	"go-feature-based-boilerplate/internal/user/dto"
	"go-feature-based-boilerplate/internal/user/usecase"
	"go-feature-based-boilerplate/pkg/contextutil"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Handler implements the gRPC UserService server interface.
type Handler struct {
	pb.UnimplementedUserServiceServer
	createUsecase *usecase.CreateUsecase
	findUsecase   *usecase.FindUsecase
	updateUsecase *usecase.UpdateUsecase
	deleteUsecase *usecase.DeleteUsecase
}

// NewHandler constructs a Handler with all required use cases.
func NewHandler(
	create *usecase.CreateUsecase,
	find *usecase.FindUsecase,
	update *usecase.UpdateUsecase,
	delete *usecase.DeleteUsecase,
) *Handler {
	return &Handler{
		createUsecase: create,
		findUsecase:   find,
		updateUsecase: update,
		deleteUsecase: delete,
	}
}

// GetUser retrieves a user by ID.
func (h *Handler) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserResponse, error) {
	user, err := h.findUsecase.FindByID(ctx, uint(req.GetId()))
	if err != nil {
		return nil, err // error interceptor handles gRPC status mapping
	}
	return &pb.GetUserResponse{User: toProto(dto.ToUserResponse(user))}, nil
}

// CreateUser creates a new user.
func (h *Handler) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.CreateUserResponse, error) {
	user, err := h.createUsecase.Execute(ctx, dto.CreateUserRequest{
		Name:     req.GetName(),
		Email:    req.GetEmail(),
		Password: req.GetPassword(),
	})
	if err != nil {
		return nil, err
	}
	return &pb.CreateUserResponse{User: toProto(dto.ToUserResponse(user))}, nil
}

// UpdateUser updates an existing user.
func (h *Handler) UpdateUser(ctx context.Context, req *pb.UpdateUserRequest) (*pb.UpdateUserResponse, error) {
	callerID, _ := contextutil.GetUserID(ctx)
	user, err := h.updateUsecase.Execute(ctx, dto.UpdateUserRequest{
		ID:       uint(req.GetId()),
		CallerID: callerID,
		Name:     req.GetName(),
	})
	if err != nil {
		return nil, err
	}
	return &pb.UpdateUserResponse{User: toProto(dto.ToUserResponse(user))}, nil
}

// DeleteUser removes a user by ID.
func (h *Handler) DeleteUser(ctx context.Context, req *pb.DeleteUserRequest) (*pb.DeleteUserResponse, error) {
	callerID, _ := contextutil.GetUserID(ctx)
	if err := h.deleteUsecase.Execute(ctx, uint(req.GetId()), callerID); err != nil {
		return nil, err
	}
	return &pb.DeleteUserResponse{}, nil
}

// RegisterGRPC registers the user service onto the provided gRPC server.
func (h *Handler) RegisterGRPC(server *grpc.Server) {
	pb.RegisterUserServiceServer(server, h)
}

// RegisterGateway registers the user service HTTP endpoints onto the gRPC-Gateway mux.
func (h *Handler) RegisterGateway(ctx context.Context, mux *runtime.ServeMux, grpcAddr string, opts []grpc.DialOption) error {
	return pb.RegisterUserServiceHandlerFromEndpoint(ctx, mux, grpcAddr, opts)
}

// toProto converts a UserResponse DTO to the generated protobuf message.
func toProto(r *dto.UserResponse) *pb.User {
	return &pb.User{
		Id:        uint64(r.ID),
		Name:      r.Name,
		Email:     r.Email,
		Role:      r.Role,
		Active:    r.Active,
		CreatedAt: timestamppb.New(r.CreatedAt),
		UpdatedAt: timestamppb.New(r.UpdatedAt),
	}
}
