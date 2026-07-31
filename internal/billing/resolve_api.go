package billing

import (
	"context"
	"fmt"

	"espx/internal/billing/pb"
	"espx/internal/config"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func OpenAPIOrDial(ctx context.Context, cfg *config.Config) (BillingAPI, func(), error) {
	noop := func() {}
	if cfg == nil || string(cfg.BillingInternalToken) == "" {
		return nil, noop, nil
	}
	token := string(cfg.BillingInternalToken)
	if !cfg.BillingGRPCEnabled {
		mod, err := OpenModule(ctx, cfg)
		if err != nil {
			return nil, noop, err
		}
		if mod == nil {
			return nil, noop, nil
		}
		return mod.API(token), mod.Close, nil
	}
	host := cfg.Billing.ServerHost
	if host == "" {
		host = "127.0.0.1"
	}
	target := host + ":" + cfg.Billing.Port
	conn, err := grpc.NewClient(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, noop, fmt.Errorf("billing gRPC dial %s: %w", target, err)
	}
	return NewGRPCBillingAPI(pb.NewBillingServiceClient(conn), token), func() { _ = conn.Close() }, nil
}
