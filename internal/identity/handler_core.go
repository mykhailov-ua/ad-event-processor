package identity

import (
	"context"
	"errors"
	"net"
	"strings"
	"time"

	"espx/internal/identity/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

const adminAPIKeyMetadata = "x-admin-api-key"

func (h *Handler) loginDuration(durationHours int32) time.Duration {
	duration := time.Duration(durationHours) * time.Hour
	if duration <= 0 {
		duration = time.Duration(h.cfg.DefaultTokenDurationHrs) * time.Hour
	} else if duration > 24*time.Hour {
		duration = 24 * time.Hour
	}
	return duration
}

func (h *Handler) loginDTO(ctx context.Context, email, password string, durationHours int32) (LoginDTO, error) {
	dto, err := h.service.Login(
		ctx,
		email,
		password,
		h.extractUserAgent(ctx),
		h.extractClientIP(ctx),
		h.loginDuration(durationHours),
	)
	if err != nil {
		return LoginDTO{}, mapError(err)
	}
	return dto, nil
}

func (h *Handler) login(ctx context.Context, email, password string, durationHours int32) (LoginResult, error) {
	dto, err := h.loginDTO(ctx, email, password, durationHours)
	if err != nil {
		return LoginResult{}, err
	}
	return loginResultFromDTO(dto), nil
}

func (h *Handler) register(ctx context.Context, email, password, role, customerID string) (RegisterResult, error) {
	cid, err := parseOptionalCustomerID(customerID)
	if err != nil {
		return RegisterResult{}, err
	}
	id, err := h.registerUser(ctx, email, password, role, cid)
	if err != nil {
		return RegisterResult{}, err
	}
	return RegisterResult{UserID: id}, nil
}

func (h *Handler) registerUser(ctx context.Context, email, password, role string, customerID uuid.UUID) (uuid.UUID, error) {
	if err := h.requireAdminKey(ctx); err != nil {
		return uuid.Nil, err
	}
	id, err := h.service.Register(ctx, RegisterDTO{
		Email:      email,
		Password:   password,
		Role:       role,
		CustomerID: customerID,
	})
	if err != nil {
		return uuid.Nil, mapError(err)
	}
	return id, nil
}

func (h *Handler) verifyTokenUser(ctx context.Context, accessToken string) (AuthUser, error) {
	user, err := h.service.VerifyToken(ctx, accessToken)
	if err != nil {
		return AuthUser{}, mapError(err)
	}
	return authUserFromDB(user), nil
}

func (h *Handler) verifyAPIKeyUser(ctx context.Context, apiKey string) (AuthUser, error) {
	user, err := h.service.VerifyAPIKey(ctx, apiKey)
	if err != nil {
		return AuthUser{}, mapError(err)
	}
	return authUserFromDB(user), nil
}

func (h *Handler) createAPIKey(ctx context.Context, name string, expiresAt *time.Time) (CreateAPIKeyResult, error) {
	user, err := h.requireAuthUser(ctx)
	if err != nil {
		return CreateAPIKeyResult{}, err
	}
	result, err := h.service.CreateAPIKey(ctx, uuidFromPg(user.ID), name, expiresAt)
	if err != nil {
		return CreateAPIKeyResult{}, mapError(err)
	}
	return result, nil
}

func (h *Handler) listAPIKeys(ctx context.Context) ([]APIKey, error) {
	user, err := h.requireAuthUser(ctx)
	if err != nil {
		return nil, err
	}
	keys, err := h.service.ListUserAPIKeys(ctx, uuidFromPg(user.ID))
	if err != nil {
		return nil, mapError(err)
	}
	return keys, nil
}

func (h *Handler) refreshSession(ctx context.Context, refreshToken string) (RefreshResult, error) {
	duration := time.Duration(h.cfg.DefaultTokenDurationHrs) * time.Hour
	accessToken, newRefresh, err := h.service.RefreshToken(ctx, refreshToken, duration)
	if err != nil {
		return RefreshResult{}, mapError(err)
	}
	return RefreshResult{
		AccessToken:  accessToken,
		RefreshToken: newRefresh,
	}, nil
}

func (h *Handler) revokeSession(ctx context.Context, refreshToken string) error {
	if err := h.service.RevokeToken(ctx, refreshToken); err != nil {
		return mapError(err)
	}
	return nil
}

func (h *Handler) changePassword(ctx context.Context, oldPassword, newPassword string) error {
	user, err := h.requireAuthUser(ctx)
	if err != nil {
		return err
	}
	if err := h.service.ChangePassword(ctx, uuidFromPg(user.ID), oldPassword, newPassword, h.extractClientIP(ctx), h.extractUserAgent(ctx)); err != nil {
		return mapError(err)
	}
	return nil
}

func (h *Handler) requestEmailVerification(ctx context.Context) (string, error) {
	user, err := h.requireAuthUser(ctx)
	if err != nil {
		return "", err
	}
	token, err := h.service.RequestEmailVerification(ctx, uuidFromPg(user.ID))
	if err != nil {
		return "", mapError(err)
	}
	return token, nil
}

func (h *Handler) confirmEmailVerification(ctx context.Context, verificationToken string) error {
	if _, err := h.service.ConfirmEmailVerification(ctx, verificationToken); err != nil {
		return mapError(err)
	}
	return nil
}

func parseOptionalCustomerID(raw string) (uuid.UUID, error) {
	if raw == "" {
		return uuid.UUID{}, nil
	}
	customerID, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, status.Error(codes.InvalidArgument, "invalid customer id")
	}
	return customerID, nil
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
