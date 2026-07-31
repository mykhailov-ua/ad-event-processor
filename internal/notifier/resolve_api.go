package notifier

import (
	"context"
	"fmt"

	"espx/internal/config"
	"espx/internal/notifier/pb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func OpenAPIOrDial(ctx context.Context, cfg *config.Config) (NotifierAPI, func(), error) {
	noop := func() {}
	if cfg == nil {
		return nil, noop, nil
	}
	if !cfg.NotifierGRPCEnabled {
		mod, err := OpenModule(ctx, cfg)
		if err != nil {
			return nil, noop, err
		}
		if mod == nil {
			return nil, noop, nil
		}
		return mod.API(), mod.Close, nil
	}
	target := cfg.Notifier.ServerHost + ":" + cfg.Notifier.Port
	conn, err := grpc.NewClient(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, noop, fmt.Errorf("notifier gRPC dial %s: %w", target, err)
	}
	return NewGRPCNotifierAPI(pb.NewNotifierServiceClient(conn)), func() { _ = conn.Close() }, nil
}
