package identity

import (
	"context"
	"time"

	"espx/internal/identity/db"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

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

func (h *Handler) verifyAPIKeyUser(ctx context.Context, apiKey string) (db.User, error) {
	user, err := h.service.VerifyAPIKey(ctx, apiKey)
	if err != nil {
		return db.User{}, mapError(err)
	}
	return user, nil
}

func (h *Handler) refreshSession(ctx context.Context, refreshToken string) (string, string, error) {
	duration := time.Duration(h.cfg.DefaultTokenDurationHrs) * time.Hour
	accessToken, newRefresh, err := h.service.RefreshToken(ctx, refreshToken, duration)
	if err != nil {
		return "", "", mapError(err)
	}
	return accessToken, newRefresh, nil
}

func (h *Handler) revokeSession(ctx context.Context, refreshToken string) error {
	if err := h.service.RevokeToken(ctx, refreshToken); err != nil {
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
