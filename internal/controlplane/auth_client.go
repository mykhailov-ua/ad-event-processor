package controlplane

import (
	"context"
	"errors"

	"espx/internal/identity"
	authpb "espx/internal/identity/pb"

	"github.com/google/uuid"
	"google.golang.org/grpc/metadata"
)

var errAuthUnavailable = errors.New("auth service not configured")

func bearerOutgoingContext(ctx context.Context, bearerToken string) context.Context {
	return metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+bearerToken)
}

type AuthClient struct {
	api identity.AuthAPI
}

func NewAuthClientFromAPI(api identity.AuthAPI) *AuthClient {
	if api == nil {
		return nil
	}
	return &AuthClient{api: api}
}

func NewAuthClient(client authpb.AuthServiceClient) *AuthClient {
	if client == nil {
		return nil
	}
	return NewAuthClientFromAPI(&grpcAuthAPI{client: client})
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

type grpcAuthAPI struct {
	client authpb.AuthServiceClient
}

func (g *grpcAuthAPI) VerifyAPIKey(ctx context.Context, apiKey string) (identity.AuthUser, error) {
	resp, err := g.client.VerifyAPIKey(ctx, &authpb.VerifyAPIKeyRequest{ApiKey: apiKey})
	if err != nil {
		return identity.AuthUser{}, err
	}
	return authUserFromPB(resp.User), nil
}

func (g *grpcAuthAPI) CreateAPIKey(ctx context.Context, bearerToken, name string) (identity.CreateAPIKeyResult, error) {
	grpcCtx := bearerOutgoingContext(ctx, bearerToken)
	resp, err := g.client.CreateAPIKey(grpcCtx, &authpb.CreateAPIKeyRequest{Name: name})
	if err != nil {
		return identity.CreateAPIKeyResult{}, err
	}
	out := identity.CreateAPIKeyResult{
		ID:     resp.Id,
		Name:   resp.Name,
		RawKey: resp.RawKey,
	}
	if resp.ExpiresAt != nil {
		t := resp.ExpiresAt.AsTime()
		out.ExpiresAt = &t
	}
	return out, nil
}

func (g *grpcAuthAPI) Login(ctx context.Context, email, password string, durationHours int32) (identity.LoginResult, error) {
	resp, err := g.client.Login(ctx, &authpb.LoginRequest{
		Email:         email,
		Password:      password,
		DurationHours: durationHours,
	})
	if err != nil {
		return identity.LoginResult{}, err
	}
	return identity.LoginResult{
		AccessToken:  resp.AccessToken,
		RefreshToken: resp.RefreshToken,
		User:         authUserFromPB(resp.User),
	}, nil
}

func (g *grpcAuthAPI) Register(ctx context.Context, adminAPIKey, email, password, role, customerID string) (identity.RegisterResult, error) {
	grpcCtx := metadata.AppendToOutgoingContext(ctx, "x-admin-api-key", adminAPIKey)
	resp, err := g.client.Register(grpcCtx, &authpb.RegisterRequest{
		Email:      email,
		Password:   password,
		Role:       role,
		CustomerId: customerID,
	})
	if err != nil {
		return identity.RegisterResult{}, err
	}
	uid, err := uuid.Parse(resp.UserId)
	if err != nil {
		return identity.RegisterResult{}, err
	}
	return identity.RegisterResult{UserID: uid}, nil
}

func (g *grpcAuthAPI) RefreshToken(ctx context.Context, refreshToken string) (identity.RefreshResult, error) {
	resp, err := g.client.RefreshToken(ctx, &authpb.RefreshTokenRequest{RefreshToken: refreshToken})
	if err != nil {
		return identity.RefreshResult{}, err
	}
	return identity.RefreshResult{
		AccessToken:  resp.AccessToken,
		RefreshToken: resp.RefreshToken,
	}, nil
}

func (g *grpcAuthAPI) RevokeToken(ctx context.Context, refreshToken string) error {
	_, err := g.client.RevokeToken(ctx, &authpb.RevokeTokenRequest{RefreshToken: refreshToken})
	return err
}

func authUserFromPB(user *authpb.User) identity.AuthUser {
	if user == nil {
		return identity.AuthUser{}
	}
	uid, _ := uuid.Parse(user.Id)
	cid, _ := uuid.Parse(user.CustomerId)
	return identity.AuthUser{
		ID:         uid,
		Email:      user.Email,
		Role:       user.Role,
		CustomerID: cid,
	}
}
