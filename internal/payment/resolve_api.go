package payment

import (
	"context"
	"fmt"

	"espx/internal/config"
	"espx/internal/payment/pb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func OpenAPIOrDial(ctx context.Context, cfg *config.Config) (PaymentAPI, func(), error) {
	noop := func() {}
	if cfg == nil || string(cfg.PaymentInternalToken) == "" {
		return nil, noop, nil
	}
	token := string(cfg.PaymentInternalToken)
	if !cfg.PaymentGRPCEnabled {
		mod, err := OpenModule(ctx, cfg)
		if err != nil {
			return nil, noop, err
		}
		if mod == nil {
			return nil, noop, nil
		}
		return mod.API(token), mod.Close, nil
	}
	host := cfg.PaymentServerHost
	if host == "" {
		host = "127.0.0.1"
	}
	target := host + ":" + cfg.PaymentServerPort
	conn, err := grpc.NewClient(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, noop, fmt.Errorf("payment gRPC dial %s: %w", target, err)
	}
	return NewGRPCPaymentAPI(pb.NewPaymentServiceClient(conn), token), func() { _ = conn.Close() }, nil
}
