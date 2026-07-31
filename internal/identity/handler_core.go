package identity

import (
	"context"
	"time"

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
