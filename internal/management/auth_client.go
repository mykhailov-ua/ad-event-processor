package management

import (
	"context"
	"errors"

	authpb "espx/internal/auth/pb"

	"google.golang.org/grpc/metadata"
)

var errAuthUnavailable = errors.New("auth service not configured")

func bearerOutgoingContext(ctx context.Context, bearerToken string) context.Context {
	return metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+bearerToken)
}

type AuthClient struct {
	client authpb.AuthServiceClient
}

func NewAuthClient(client authpb.AuthServiceClient) *AuthClient {
	if client == nil {
		return nil
	}
	return &AuthClient{client: client}
}

func (c *AuthClient) VerifyAPIKey(ctx context.Context, apiKey string) (*authpb.VerifyAPIKeyResponse, error) {
	if c == nil || c.client == nil {
		return nil, errAuthUnavailable
	}
	return c.client.VerifyAPIKey(ctx, &authpb.VerifyAPIKeyRequest{ApiKey: apiKey})
}

func (c *AuthClient) CreateAPIKey(ctx context.Context, bearerToken, name string) (*authpb.CreateAPIKeyResponse, error) {
	if c == nil || c.client == nil {
		return nil, errAuthUnavailable
	}
	grpcCtx := bearerOutgoingContext(ctx, bearerToken)
	return c.client.CreateAPIKey(grpcCtx, &authpb.CreateAPIKeyRequest{Name: name})
}
