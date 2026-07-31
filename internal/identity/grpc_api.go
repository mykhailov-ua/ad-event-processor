package identity

import (
	"context"

	"espx/internal/identity/pb"

	"github.com/google/uuid"
	"google.golang.org/grpc/metadata"
)

type grpcAuthAPI struct {
	client pb.AuthServiceClient
}

func NewGRPCAuthAPI(client pb.AuthServiceClient) AuthAPI {
	if client == nil {
		return nil
	}
	return &grpcAuthAPI{client: client}
}

func bearerOutgoingContext(ctx context.Context, bearerToken string) context.Context {
	return metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+bearerToken)
}

func (g *grpcAuthAPI) VerifyAPIKey(ctx context.Context, apiKey string) (AuthUser, error) {
	resp, err := g.client.VerifyAPIKey(ctx, &pb.VerifyAPIKeyRequest{ApiKey: apiKey})
	if err != nil {
		return AuthUser{}, err
	}
	return authUserFromPB(resp.User), nil
}

func (g *grpcAuthAPI) CreateAPIKey(ctx context.Context, bearerToken, name string) (CreateAPIKeyResult, error) {
	grpcCtx := bearerOutgoingContext(ctx, bearerToken)
	resp, err := g.client.CreateAPIKey(grpcCtx, &pb.CreateAPIKeyRequest{Name: name})
	if err != nil {
		return CreateAPIKeyResult{}, err
	}
	out := CreateAPIKeyResult{
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

func (g *grpcAuthAPI) Login(ctx context.Context, email, password string, durationHours int32) (LoginResult, error) {
	resp, err := g.client.Login(ctx, &pb.LoginRequest{
		Email:         email,
		Password:      password,
		DurationHours: durationHours,
	})
	if err != nil {
		return LoginResult{}, err
	}
	if resp == nil {
		return LoginResult{}, nil
	}
	return loginResultFromPB(*resp), nil
}

func (g *grpcAuthAPI) Register(ctx context.Context, adminAPIKey, email, password, role, customerID string) (RegisterResult, error) {
	grpcCtx := metadata.AppendToOutgoingContext(ctx, "x-admin-api-key", adminAPIKey)
	resp, err := g.client.Register(grpcCtx, &pb.RegisterRequest{
		Email:      email,
		Password:   password,
		Role:       role,
		CustomerId: customerID,
	})
	if err != nil {
		return RegisterResult{}, err
	}
	uid, err := uuid.Parse(resp.UserId)
	if err != nil {
		return RegisterResult{}, err
	}
	return RegisterResult{UserID: uid}, nil
}

func (g *grpcAuthAPI) RefreshToken(ctx context.Context, refreshToken string) (RefreshResult, error) {
	resp, err := g.client.RefreshToken(ctx, &pb.RefreshTokenRequest{RefreshToken: refreshToken})
	if err != nil {
		return RefreshResult{}, err
	}
	return RefreshResult{
		AccessToken:  resp.AccessToken,
		RefreshToken: resp.RefreshToken,
	}, nil
}

func (g *grpcAuthAPI) RevokeToken(ctx context.Context, refreshToken string) error {
	_, err := g.client.RevokeToken(ctx, &pb.RevokeTokenRequest{RefreshToken: refreshToken})
	return err
}

func authUserFromPB(user *pb.User) AuthUser {
	if user == nil {
		return AuthUser{}
	}
	uid, _ := uuid.Parse(user.Id)
	cid, _ := uuid.Parse(user.CustomerId)
	return AuthUser{
		ID:         uid,
		Email:      user.Email,
		Role:       user.Role,
		CustomerID: cid,
	}
}
