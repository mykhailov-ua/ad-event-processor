package controlplane

import (
	"context"
	"errors"

	"espx/internal/identity"
)

var errAuthUnavailable = errors.New("auth service not configured")

type AuthClient struct {
	api identity.AuthAPI
}

func NewAuthClientFromAPI(api identity.AuthAPI) *AuthClient {
	if api == nil {
		return nil
	}
	return &AuthClient{api: api}
}

func (c *AuthClient) VerifyAPIKey(ctx context.Context, apiKey string) (identity.AuthUser, error) {
	if c == nil || c.api == nil {
		return identity.AuthUser{}, errAuthUnavailable
	}
	return c.api.VerifyAPIKey(ctx, apiKey)
}

func (c *AuthClient) CreateAPIKey(ctx context.Context, bearerToken, name string) (identity.CreateAPIKeyResult, error) {
	if c == nil || c.api == nil {
		return identity.CreateAPIKeyResult{}, errAuthUnavailable
	}
	return c.api.CreateAPIKey(ctx, bearerToken, name)
}

func (c *AuthClient) Login(ctx context.Context, email, password string, durationHours int32) (identity.LoginResult, error) {
	if c == nil || c.api == nil {
		return identity.LoginResult{}, errAuthUnavailable
	}
	return c.api.Login(ctx, email, password, durationHours)
}

func (c *AuthClient) Register(ctx context.Context, adminAPIKey, email, password, role, customerID string) (identity.RegisterResult, error) {
	if c == nil || c.api == nil {
		return identity.RegisterResult{}, errAuthUnavailable
	}
	return c.api.Register(ctx, adminAPIKey, email, password, role, customerID)
}

func (c *AuthClient) RefreshToken(ctx context.Context, refreshToken string) (identity.RefreshResult, error) {
	if c == nil || c.api == nil {
		return identity.RefreshResult{}, errAuthUnavailable
	}
	return c.api.RefreshToken(ctx, refreshToken)
}

func (c *AuthClient) RevokeToken(ctx context.Context, refreshToken string) error {
	if c == nil || c.api == nil {
		return errAuthUnavailable
	}
	return c.api.RevokeToken(ctx, refreshToken)
}
