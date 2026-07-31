package controlplane

import (
	"context"
	"fmt"
	"os"

	"espx/internal/config"
	"espx/internal/identity"
	auth_pb "espx/internal/identity/pb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func TryAuthClient(ctx context.Context, cfg *config.Config) (*AuthClient, func(), error) {
	noop := func() {}
	if cfg == nil {
		return nil, noop, nil
	}
	if !cfg.AuthGRPCEnabled {
		mod, err := identity.OpenModule(ctx, cfg)
		if err != nil {
			return nil, noop, err
		}
		if mod == nil {
			return nil, noop, nil
		}
		return NewAuthClientFromAPI(mod.API()), mod.Close, nil
	}

	target := "127.0.0.1:" + cfg.AuthServerPort
	if host := os.Getenv("AUTH_SERVER_HOST"); host != "" {
		target = host + ":" + cfg.AuthServerPort
	}
	conn, err := grpc.NewClient(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, noop, fmt.Errorf("auth gRPC dial %s: %w", target, err)
	}
	return NewAuthClient(auth_pb.NewAuthServiceClient(conn)), func() { _ = conn.Close() }, nil
}
