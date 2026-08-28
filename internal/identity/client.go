package identity

import (
	"context"
	"errors"
)

var ErrAuthUnavailable = errors.New("auth service not configured")

type AuthClient struct {
	api AuthAPI
}

func NewAuthClientFromAPI(api AuthAPI) *AuthClient {
	if api == nil {
		return nil
	}
	return &AuthClient{api: api}
}

func (c *AuthClient) VerifyAPIKey(ctx context.Context, apiKey string) (AuthUser, error) {
	if c == nil || c.api == nil {
		return AuthUser{}, ErrAuthUnavailable
	}
	return c.api.VerifyAPIKey(ctx, apiKey)
}

func (c *AuthClient) CreateAPIKey(ctx context.Context, bearerToken, name string, scopes []string) (CreateAPIKeyResult, error) {
	if c == nil || c.api == nil {
		return CreateAPIKeyResult{}, ErrAuthUnavailable
	}
	return c.api.CreateAPIKey(ctx, bearerToken, name, scopes)
}

func (c *AuthClient) Login(ctx context.Context, email, password string, durationHours int32) (LoginResult, error) {
	if c == nil || c.api == nil {
		return LoginResult{}, ErrAuthUnavailable
	}
	return c.api.Login(ctx, email, password, durationHours)
}

func (c *AuthClient) Register(ctx context.Context, adminAPIKey, email, password, role, customerID string) (RegisterResult, error) {
	if c == nil || c.api == nil {
		return RegisterResult{}, ErrAuthUnavailable
	}
	return c.api.Register(ctx, adminAPIKey, email, password, role, customerID)
}

func (c *AuthClient) RefreshToken(ctx context.Context, refreshToken string) (RefreshResult, error) {
	if c == nil || c.api == nil {
		return RefreshResult{}, ErrAuthUnavailable
	}
	return c.api.RefreshToken(ctx, refreshToken)
}

func (c *AuthClient) RevokeToken(ctx context.Context, refreshToken string) error {
	if c == nil || c.api == nil {
		return ErrAuthUnavailable
	}
	return c.api.RevokeToken(ctx, refreshToken)
}
