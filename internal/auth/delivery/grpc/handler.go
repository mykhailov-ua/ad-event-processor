package grpc

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/mykhailov-ua/ad-event-processor/internal/auth"
	"github.com/mykhailov-ua/ad-event-processor/internal/auth/pb"
	"github.com/mykhailov-ua/ad-event-processor/internal/config"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Handler struct {
	pb.UnimplementedAuthServiceServer
	service *auth.Service
	cfg     *config.Config
}

func NewHandler(service *auth.Service, cfg *config.Config) *Handler {
	return &Handler{
		service: service,
		cfg:     cfg,
	}
}

func (h *Handler) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	customerID, _ := uuid.Parse(req.Customer_Id)
	id, err := h.service.Register(ctx, auth.RegisterRequest{
		Email:      req.Email,
		Password:   req.Password,
		Role:       req.Role,
		CustomerID: customerID,
	})
	if err != nil {
		return nil, err
	}

	return &pb.RegisterResponse{
		UserId: id.String(),
	}, nil
}

func (h *Handler) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	duration := time.Duration(req.Duration_Hours) * time.Hour
	if duration == 0 {
		duration = time.Duration(h.cfg.DefaultTokenDurationHrs) * time.Hour
	}

	resp, err := h.service.Login(ctx, req.Email, req.Password, duration)
	if err != nil {
		return nil, err
	}

	return &pb.LoginResponse{
		AccessToken: resp.AccessToken,
		User: &pb.User{
			Id:         resp.User.ID.String(),
			Email:      resp.User.Email,
			Role:       resp.User.Role,
			CustomerId: resp.User.CustomerID.String(),
			CreatedAt:  timestamppb.New(resp.User.CreatedAt.Time),
		},
	}, nil
}

func (h *Handler) VerifyToken(ctx context.Context, req *pb.VerifyTokenRequest) (*pb.VerifyTokenResponse, error) {
	return nil, nil
}
