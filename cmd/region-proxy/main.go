package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"time"

	"github.com/bidshard/ad-event-processor/internal/config"
	_ "github.com/bidshard/ad-event-processor/internal/metrics"
	"github.com/bidshard/ad-event-processor/pkg/broker/server"
	"github.com/bidshard/ad-event-processor/pkg/iogate"
	"github.com/bidshard/ad-event-processor/pkg/lifecycle"
	"github.com/bidshard/ad-event-processor/pkg/regionproxy/keygen"
	"github.com/bidshard/ad-event-processor/pkg/regionproxy/opkey"
	rserver "github.com/bidshard/ad-event-processor/pkg/regionproxy/server"
	"github.com/bidshard/ad-event-processor/pkg/regionproxy/uplink"

	"github.com/redis/go-redis/v9"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:9093", "gnet TCP ingress address")
	healthAddr := flag.String("health-addr", "127.0.0.1:8082", "HTTP health and metrics address")
	dataDir := flag.String("data-dir", "/tmp/ad-event-processor-region-proxy", "WAL data directory")
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
		defer func() { _ = rdb.Close() }()
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
	coord.Start(context.Background())

	srv.LogStart()
	sig := lifecycle.WaitSignal()
	slog.Info("region-proxy shutting down", "signal", sig.String())
	// coord first (HA deregister), then gnet + metrics HTTP drain.
	coord.Stop()
	srv.Stop()
}
