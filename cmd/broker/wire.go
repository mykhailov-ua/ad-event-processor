// Broker serve wiring: mmap WAL server + Redis coordinator lifecycle.
//
// WAL layout (pkg/broker/log, -data-dir):
//   - Partition dir holds paired segments: %020d.log (records) + %020d.index (sparse offset index).
//   - Record: 4-byte BE length (includes 8-byte offset) + 8-byte BE offset + payload (mmap append on leader).
//   - Index entry: 16 bytes (offset uint64 BE + log position uint64 BE) every -index-kb stride.
//   - Segment rolls at -max-seg-mb; fencing.epoch file bumps on leader demotion (stale epoch reject).
//
// Coordinator Redis HA (internal/broker.Coordinator, -redis-url):
//   - Per topic: SETNX ad_event_processor:topics:<topic>:leader lease; INCR leader_epoch on claim.
//   - Followers replicate via gnet Fetch; leader PublishLogHWM; Sentinel via BROKER_REDIS_SENTINEL_*.
//   - Fail-closed produce when not leader-ready; step down on lease renew failures.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	broker "ad-event-processor/internal/broker"
	"ad-event-processor/internal/config"
	"ad-event-processor/pkg/broker/log"
	"ad-event-processor/pkg/lifecycle"
)

// serveWireConfig is the resolved serve flags/env surface for wireAndRunServe.
type serveWireConfig struct {
	Addr          string // gnet Produce/Fetch listen (TCP or unix)
	HealthAddr    string // HTTP /health and /metrics
	DataDir       string // WAL segment directory
	NodeID        string // coordinator node id
	RedisURL      string // HA leadership and offset metadata
	MaxSegBytes   int64  // max WAL segment size bytes (-max-seg-mb * 1024^2)
	IndexInterval int64  // sparse index stride bytes (-index-kb * 1024)
}

func wireAndRunServe(cfg serveWireConfig) {
	srv := broker.NewServer(cfg.Addr, cfg.DataDir, cfg.MaxSegBytes, cfg.IndexInterval)
	srv.SetHealthAddr(cfg.HealthAddr)
	srv.SetShutdownTimeout(config.LifecycleShutdownTimeout())

	coord, err := broker.NewCoordinator(cfg.NodeID, cfg.Addr, cfg.RedisURL, srv)
	if err != nil {
		slog.Error("broker coordinator init failed", "error", err)
		os.Exit(1)
	}
	srv.SetCoordinator(coord)
	srv.SetDurability(log.DefaultDurabilityConfig())

	ctx, stop := lifecycle.NotifyContext(context.Background())
	coord.Start(ctx)

	if err := srv.Start(); err != nil {
		slog.Error("broker server start failed", "error", err)
		stop()
		os.Exit(1)
	}

	slog.Info("broker listening",
		"addr", srv.Addr(),
		"health_addr", srv.HealthAddr(),
		"data_dir", cfg.DataDir,
	)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig

	srv.Stop()
	coord.Stop()
	stop()
	slog.Info("broker shutdown complete")
}
