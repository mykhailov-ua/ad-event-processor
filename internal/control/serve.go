package control

import (
	"context"
	"log/slog"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/controlplane"
	"ad-event-processor/internal/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)

// StartControlServers: optional UDP/TCP control plane to push quota epoch and slot map to trackers.
func StartControlServers(
	ctx context.Context,
	cfg *config.Config,
	pool *pgxpool.Pool,
	sharder domain.Sharder,
	numShards int,
) (controlplane.TCPControlPublisher, func(), error) {
	if cfg == nil {
		return nil, nil, nil
	}
	var closers []func()
	closeAll := func() {
		for i := len(closers) - 1; i >= 0; i-- {
			closers[i]()
		}
	}

	if cfg.UDPControlEnabled {
		udpSrv := NewUDPControlServer(cfg, pool, sharder, numShards)
		if err := udpSrv.Start(ctx); err != nil {
			slog.Error("udp control server start failed", "error", err)
			closeAll()
			return nil, nil, err
		}
		closers = append(closers, func() { _ = udpSrv.Close() })
	}

	var tcpSrv *TCPControlServer
	if cfg.TCPControlEnabled {
		tcpSrv = NewTCPControlServer(cfg, pool, sharder, numShards)
		if err := tcpSrv.Start(ctx); err != nil {
			slog.Error("tcp control server start failed", "error", err)
			closeAll()
			return nil, nil, err
		}
		closers = append(closers, func() { _ = tcpSrv.Close() })
	}

	if len(closers) == 0 {
		return nil, nil, nil
	}
	return tcpSrv, closeAll, nil
}
