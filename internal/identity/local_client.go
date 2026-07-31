package identity

import (
	"context"

	"espx/internal/identity/pb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type localGRPCClient struct {
	handler pb.AuthServiceServer
}

func NewLocalGRPCClient(handler pb.AuthServiceServer) pb.AuthServiceClient {
	if handler == nil {
		return nil
	}
	return &localGRPCClient{handler: handler}
}

func bridgeOutgoing(ctx context.Context) context.Context {
	if md, ok := metadata.FromOutgoingContext(ctx); ok {
		return metadata.NewIncomingContext(ctx, md)
	}
	return ctx
}

func (c *localGRPCClient) Register(ctx context.Context, in *pb.RegisterRequest, _ ...grpc.CallOption) (*pb.RegisterResponse, error) {
	return c.handler.Register(bridgeOutgoing(ctx), in)
}

func (c *localGRPCClient) Login(ctx context.Context, in *pb.LoginRequest, _ ...grpc.CallOption) (*pb.LoginResponse, error) {
	return c.handler.Login(bridgeOutgoing(ctx), in)
}

func (c *localGRPCClient) VerifyToken(ctx context.Context, in *pb.VerifyTokenRequest, _ ...grpc.CallOption) (*pb.VerifyTokenResponse, error) {
	return c.handler.VerifyToken(bridgeOutgoing(ctx), in)
}

func (c *localGRPCClient) RefreshToken(ctx context.Context, in *pb.RefreshTokenRequest, _ ...grpc.CallOption) (*pb.RefreshTokenResponse, error) {
	return c.handler.RefreshToken(bridgeOutgoing(ctx), in)
}

func (c *localGRPCClient) RevokeToken(ctx context.Context, in *pb.RevokeTokenRequest, _ ...grpc.CallOption) (*pb.RevokeTokenResponse, error) {
	return c.handler.RevokeToken(bridgeOutgoing(ctx), in)
}

func (c *localGRPCClient) CreateAPIKey(ctx context.Context, in *pb.CreateAPIKeyRequest, _ ...grpc.CallOption) (*pb.CreateAPIKeyResponse, error) {
	return c.handler.CreateAPIKey(bridgeOutgoing(ctx), in)
}

func (c *localGRPCClient) VerifyAPIKey(ctx context.Context, in *pb.VerifyAPIKeyRequest, _ ...grpc.CallOption) (*pb.VerifyAPIKeyResponse, error) {
	return c.handler.VerifyAPIKey(bridgeOutgoing(ctx), in)
}

func (c *localGRPCClient) ListAPIKeys(ctx context.Context, in *pb.ListAPIKeysRequest, _ ...grpc.CallOption) (*pb.ListAPIKeysResponse, error) {
	return c.handler.ListAPIKeys(bridgeOutgoing(ctx), in)
}

func (c *localGRPCClient) ChangePassword(ctx context.Context, in *pb.ChangePasswordRequest, _ ...grpc.CallOption) (*pb.ChangePasswordResponse, error) {
	return c.handler.ChangePassword(bridgeOutgoing(ctx), in)
}

func (c *localGRPCClient) RequestEmailVerification(ctx context.Context, in *pb.RequestEmailVerificationRequest, _ ...grpc.CallOption) (*pb.RequestEmailVerificationResponse, error) {
	return c.handler.RequestEmailVerification(bridgeOutgoing(ctx), in)
}

func (c *localGRPCClient) ConfirmEmailVerification(ctx context.Context, in *pb.ConfirmEmailVerificationRequest, _ ...grpc.CallOption) (*pb.ConfirmEmailVerificationResponse, error) {
	return c.handler.ConfirmEmailVerification(bridgeOutgoing(ctx), in)
}
