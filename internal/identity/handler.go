package identity

import (
	"context"
	"errors"
	"net"
	"strings"

	"espx/internal/config"
	"espx/internal/identity/db"
	"espx/internal/identity/pb"

	"github.com/jackc/pgx/v5"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

const adminAPIKeyMetadata = "x-admin-api-key"

type Handler struct {
	pb.UnimplementedAuthServiceServer
	service *Service
	cfg     *config.Config
}

func NewHandler(service *Service, cfg *config.Config) *Handler {
	return &Handler{
		service: service,
		cfg:     cfg,
	}
}

func (h *Handler) extractClientIP(ctx context.Context) string {
	peerIP := "unknown"
	if p, ok := peer.FromContext(ctx); ok {
		host, _, err := net.SplitHostPort(p.Addr.String())
		if err == nil {
			peerIP = host
		} else {
			peerIP = p.Addr.String()
		}
	}

	isTrusted := false
	for _, tp := range h.cfg.TrustedProxies {
		if tp != "" && peerIP == tp {
			isTrusted = true
			break
		}
	}

	if peerIP == "127.0.0.1" || peerIP == "::1" || peerIP == "bufconn" {
		isTrusted = true
	}

	if isTrusted {
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			if xri := md.Get("x-real-ip"); len(xri) > 0 && xri[0] != "" {
				return strings.TrimSpace(xri[0])
			}
			if xff := md.Get("x-forwarded-for"); len(xff) > 0 {
				ips := strings.Split(xff[0], ",")
				if len(ips) > 0 {
					val := strings.TrimSpace(ips[len(ips)-1])
					if val != "" {
						return val
					}
				}
			}
		}
	}

	return peerIP
}

func (h *Handler) extractUserAgent(ctx context.Context) string {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if ua := md.Get("user-agent"); len(ua) > 0 {
			return ua[0]
		}
	}
	return "grpc-client"
}

func (h *Handler) requireAdminKey(ctx context.Context) error {
	if h.cfg == nil || h.cfg.AdminAPIKey == "" {
		return status.Error(codes.PermissionDenied, "admin credentials not configured")
	}
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return status.Error(codes.Unauthenticated, "missing credentials")
	}
	keys := md.Get(adminAPIKeyMetadata)
	if len(keys) == 0 || keys[0] == "" || keys[0] != string(h.cfg.AdminAPIKey) {
		return status.Error(codes.PermissionDenied, "admin credentials required")
	}
	return nil
}

func (h *Handler) requireAuthUser(ctx context.Context) (db.User, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return db.User{}, status.Error(codes.Unauthenticated, "missing authorization metadata")
	}
	values := md.Get(authorizationHeaderKey)
	if len(values) == 0 {
		return db.User{}, status.Error(codes.Unauthenticated, "authorization header is not provided")
	}
	accessToken, ok := parseBearerToken(values[0])
	if !ok {
		return db.User{}, status.Error(codes.Unauthenticated, "invalid authorization header format")
	}
	user, err := h.service.VerifyToken(ctx, accessToken)
	if err != nil {
		return db.User{}, mapError(err)
	}
	return user, nil
}

func mapError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrRateLimitExceeded) {
		return status.Error(codes.ResourceExhausted, err.Error())
	}
	if errors.Is(err, ErrInvalidCredentials) || errors.Is(err, ErrInvalidToken) || errors.Is(err, ErrExpiredToken) || errors.Is(err, ErrAccountLocked) || errors.Is(err, ErrSessionBlocked) || errors.Is(err, ErrEmailNotVerified) || errors.Is(err, ErrInvalidAPIKey) {
		return status.Error(codes.Unauthenticated, err.Error())
	}
	if errors.Is(err, ErrUserAlreadyExists) {
		return status.Error(codes.AlreadyExists, err.Error())
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return status.Error(codes.NotFound, "user not found")
	}
	if errors.Is(err, ErrValidation) || errors.Is(err, ErrPasswordReuse) {
		return status.Error(codes.InvalidArgument, err.Error())
	}
	return status.Error(codes.Internal, "internal server error")
}
