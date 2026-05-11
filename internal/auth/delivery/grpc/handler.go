package grpc

import (
	"context"
	"errors"
	"strings"
	"time"

	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/mykhailov-ua/ad-event-processor/internal/auth"
	"github.com/mykhailov-ua/ad-event-processor/internal/auth/pb"
	"github.com/mykhailov-ua/ad-event-processor/internal/auth/token"
	"github.com/mykhailov-ua/ad-event-processor/internal/config"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

func extractClientIP(ctx context.Context) string {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if xff := md.Get("x-forwarded-for"); len(xff) > 0 {
			// Extract the first IP if there is a comma-separated list
			ips := strings.Split(xff[0], ",")
			if len(ips) > 0 {
				return strings.TrimSpace(ips[0])
			}
		}
		if xri := md.Get("x-real-ip"); len(xri) > 0 {
			return xri[0]
		}
	}

	if p, ok := peer.FromContext(ctx); ok {
		return p.Addr.String()
	}
	return "unknown"
}

func (h *Handler) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	customerID, _ := uuid.Parse(req.CustomerId)
	id, err := h.service.Register(ctx, auth.RegisterRequest{
		Email:      req.Email,
		Password:   req.Password,
		Role:       req.Role,
		CustomerID: customerID,
	})
	if err != nil {
		return nil, mapError(err)
	}

	return &pb.RegisterResponse{
		UserId: id.String(),
	}, nil
}

func (h *Handler) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	duration := time.Duration(req.DurationHours) * time.Hour
	if duration == 0 {
		duration = time.Duration(h.cfg.DefaultTokenDurationHrs) * time.Hour
	}

	clientIP := extractClientIP(ctx)

	userAgent := "grpc-client"
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if ua := md.Get("user-agent"); len(ua) > 0 {
			userAgent = ua[0]
		}
	}

	resp, err := h.service.Login(ctx, req.Email, req.Password, userAgent, clientIP, duration)
	if err != nil {
		return nil, mapError(err)
	}

	return &pb.LoginResponse{
		AccessToken:  resp.AccessToken,
		RefreshToken: resp.RefreshToken,
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
	user, err := h.service.VerifyToken(ctx, req.AccessToken)
	if err != nil {
		return nil, mapError(err)
	}

	return &pb.VerifyTokenResponse{
		User: &pb.User{
			Id:         user.ID.String(),
			Email:      user.Email,
			Role:       user.Role,
			CustomerId: user.CustomerID.String(),
			CreatedAt:  timestamppb.New(user.CreatedAt.Time),
		},
	}, nil
}

func (h *Handler) RefreshToken(ctx context.Context, req *pb.RefreshTokenRequest) (*pb.RefreshTokenResponse, error) {
	duration := time.Duration(h.cfg.DefaultTokenDurationHrs) * time.Hour
	accessToken, refreshToken, err := h.service.RefreshToken(ctx, req.RefreshToken, duration)
	if err != nil {
		return nil, mapError(err)
	}
	return &pb.RefreshTokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (h *Handler) RevokeToken(ctx context.Context, req *pb.RevokeTokenRequest) (*pb.RevokeTokenResponse, error) {
	err := h.service.RevokeToken(ctx, req.RefreshToken)
	if err != nil {
		return nil, mapError(err)
	}
	return &pb.RevokeTokenResponse{}, nil
}

func mapError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, auth.ErrInvalidCredentials) || errors.Is(err, token.ErrInvalidToken) || errors.Is(err, token.ErrExpiredToken) || errors.Is(err, auth.ErrAccountLocked) || errors.Is(err, auth.ErrSessionBlocked) {
		return status.Errorf(codes.Unauthenticated, "%v", err)
	}
	if errors.Is(err, auth.ErrUserAlreadyExists) {
		return status.Errorf(codes.AlreadyExists, "%v", err)
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return status.Errorf(codes.NotFound, "user not found")
	}
	if errors.Is(err, auth.ErrValidation) {
		return status.Errorf(codes.InvalidArgument, "%v", err)
	}
	return status.Errorf(codes.Internal, "internal server error: %v", err)
}
