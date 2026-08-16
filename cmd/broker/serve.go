package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bidshard/ad-event-processor/internal/config"
	"github.com/bidshard/ad-event-processor/pkg/broker/log"
	"github.com/bidshard/ad-event-processor/pkg/broker/server"
	"github.com/bidshard/ad-event-processor/pkg/lifecycle"
	"github.com/bidshard/ad-event-processor/pkg/netaddr"
	"github.com/bidshard/ad-event-processor/pkg/runtimepaths"
)

func runServe(args []string) {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	addr := fs.String("addr", "", "gnet listen address (tcp host:port or unix socket path)")
	healthAddr := fs.String("health-addr", "", "HTTP health/metrics address")
	dataDir := fs.String("data-dir", "/var/lib/ad-event-processor/broker", "WAL data directory")
	nodeID := fs.String("node-id", "broker-1", "Broker node ID")
	redisURL := fs.String("redis-url", "", "Redis URL or unix socket path for HA coordination")
	maxSegMB := fs.Int("max-seg-mb", 64, "Max segment size in MB")
	indexKB := fs.Int("index-kb", 4, "Index interval in KB")
	_ = fs.Parse(args)

	if *addr == "" {
		*addr = netaddr.ResolveListenAddr("127.0.0.1:9092", runtimepaths.BrokerGnetSocket())
	}
	if *healthAddr == "" {
		*healthAddr = netaddr.ResolveListenAddr("127.0.0.1:8084", runtimepaths.BrokerHealthSocket())
	}
	if *redisURL == "" {
		*redisURL = os.Getenv("BROKER_REDIS_URL")
	}
	if *redisURL == "" {
		if raw := os.Getenv("REDIS_ADDRS"); raw != "" {
			first := raw
			if i := strings.Index(raw, ","); i >= 0 {
				first = strings.TrimSpace(raw[:i])
			} else {
				first = strings.TrimSpace(raw)
			}
			*redisURL = netaddr.RedisURLFromAddr(first, os.Getenv("REDIS_PASSWORD"), 0)
		}
	}

	maxSeg := int64(*maxSegMB) * 1024 * 1024
	indexInterval := int64(*indexKB) * 1024

	srv := server.NewServer(*addr, *dataDir, maxSeg, indexInterval)
	srv.SetHealthAddr(*healthAddr)
	srv.SetShutdownTimeout(config.LifecycleShutdownTimeout())

	coord, err := server.NewCoordinator(*nodeID, *addr, *redisURL, srv)
	if err != nil {
		slog.Error("broker coordinator init failed", "error", err)
		os.Exit(1)
	}
	srv.SetCoordinator(coord)
	srv.SetDurability(log.DefaultDurabilityConfig())

	ctx, stop := lifecycle.NotifyContext(context.Background())
	defer stop()
	coord.Start(ctx)

	if err := srv.Start(); err != nil {
		slog.Error("broker server start failed", "error", err)
		os.Exit(1)
	}

	slog.Info("broker listening",
		"addr", srv.Addr(),
		"health_addr", srv.HealthAddr(),
		"data_dir", *dataDir,
	)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig

	srv.Stop()
	coord.Stop()
	slog.Info("broker shutdown complete")
}
