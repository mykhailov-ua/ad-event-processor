package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"time"

	"espx/internal/config"
	_ "espx/internal/metrics"
	"espx/pkg/broker/server"
	"espx/pkg/iogate"
	"espx/pkg/lifecycle"
	"espx/pkg/regionproxy/keygen"
	"espx/pkg/regionproxy/opkey"
	rserver "espx/pkg/regionproxy/server"
	"espx/pkg/regionproxy/uplink"

	"github.com/redis/go-redis/v9"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:9093", "gnet TCP ingress address")
	healthAddr := flag.String("health-addr", "127.0.0.1:8082", "HTTP health and metrics address")
	dataDir := flag.String("data-dir", "/tmp/espx-region-proxy", "WAL data directory")
	nodeID := flag.String("node-id", "region-proxy-1", "Unique node ID")
	regionCode := flag.Uint("region-code", 1, "Region code for dedup scope")
	redisURL := flag.String("redis-url", "redis://127.0.0.1:6379/0", "Redis URL for HA coordination")
	globalIngestURL := flag.String("global-ingest-url", os.Getenv("GLOBAL_INGEST_URL"), "Global management ingest URL")
	globalIngestAPIKey := flag.String("global-ingest-api-key", os.Getenv("GLOBAL_INGEST_API_KEY"), "API key for global ingest")
	flag.Parse()

	gateCfg := iogate.DefaultConfig()
	gate := iogate.NewDiskWriteGate(gateCfg)

	srv, err := rserver.NewServer(*addr, *dataDir, gate)
	if err != nil {
		slog.Error("failed to start region-proxy", "error", err)
		os.Exit(1)
	}
	defer srv.Stop()
	srv.SetHealthAddr(*healthAddr)
	srv.SetKeyGen(keygen.Config{
		RegionCode:   uint8(*regionCode),
		NodeID:       *nodeID,
		PollInterval: time.Millisecond,
		BatchSize:    256,
	})
	srv.SetOpKey(opkey.Config{
		NodeID:       *nodeID,
		PollInterval: time.Millisecond,
		BatchSize:    256,
		Watermark:    1000,
	})
	if *globalIngestURL != "" {
		srv.SetUplink(uplink.Config{
			RegionCode:   uint8(*regionCode),
			NodeID:       *nodeID,
			GlobalURL:    *globalIngestURL,
			APIKey:       *globalIngestAPIKey,
			PollInterval: time.Millisecond,
			BatchSize:    64,
		})
	}
	srv.SetShutdownTimeout(config.LifecycleShutdownTimeout())
	srv.SetReadyProbe(func(ctx context.Context) error {
		opts, err := redis.ParseURL(*redisURL)
		if err != nil {
			return err
		}
		rdb := redis.NewClient(opts)
		defer rdb.Close()
		return rdb.Ping(ctx).Err()
	})

	if err := srv.Start(); err != nil {
		slog.Error("region-proxy gnet start failed", "error", err)
		os.Exit(1)
	}

	coord, err := server.NewCoordinator(*nodeID, srv.Addr(), *redisURL, srv)
	if err != nil {
		slog.Error("region-proxy coordinator failed", "error", err)
		os.Exit(1)
	}
	srv.SetCoordinator(coord)
	coord.Start()

	srv.LogStart()
	sig := lifecycle.WaitSignal()
	slog.Info("region-proxy shutting down", "signal", sig.String())
	coord.Stop()
}
