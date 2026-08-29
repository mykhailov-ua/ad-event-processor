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

type serveWireConfig struct {
	Addr          string
	HealthAddr    string
	DataDir       string
	NodeID        string
	RedisURL      string
	MaxSegBytes   int64
	IndexInterval int64
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
