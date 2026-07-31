package payment

import (
	"context"
	"fmt"

	"espx/internal/config"
	"espx/internal/controlplane/pb"
	"espx/internal/domain"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func OpenSettlementAPIOrDial(ctx context.Context, cfg *config.Config) (domain.PaymentSettlement, func(), error) {
	noop := func() {}
	if cfg == nil {
		return nil, noop, nil
	}
	if !cfg.SettlementGRPCEnabled {
		return nil, noop, nil
	}
	target := cfg.SettlementServerHost + ":" + cfg.SettlementServerPort
	conn, err := grpc.NewClient(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, noop, fmt.Errorf("dial settlement %s: %w", target, err)
	}
	api := newGRPCSettlementClient(pb.NewSettlementServiceClient(conn), string(cfg.SettlementInternalToken))
	return api, func() { _ = conn.Close() }, nil
}
