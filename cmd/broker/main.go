package main

import (
	"flag"
	"log/slog"
	"os"
	"time"

	"espx/internal/config"
	"espx/pkg/branding"
	blog "espx/pkg/broker/log"
	"espx/pkg/broker/server"
	"espx/pkg/lifecycle"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:9092", "Address for gnet TCP traffic")
	healthAddr := flag.String("health-addr", "127.0.0.1:8081", "Address for HTTP health checks")
	dataDir := flag.String("data-dir", "/tmp/espx-broker", "Data directory for segments")
	nodeID := flag.String("node-id", "broker-1", "Unique node ID")
	redisURL := flag.String("redis-url", "redis://127.0.0.1:6379/0", "Redis URL for coordination")
	maxSegSize := flag.Int64("max-seg-size", 64*1024*1024, "Maximum segment size in bytes")
	indexInterval := flag.Int64("index-interval", 4096, "Index interval in bytes")
	durabilityMode := flag.String("durability", "async", "Durability mode: async|group|sync")
	flushInterval := flag.Duration("flush-interval", 100*time.Millisecond, "Background fsync interval for async/group durability")
	groupCommitRecords := flag.Int64("group-commit-records", 64, "Records per group commit before fsync")
	flag.Parse()

	mode, err := blog.ParseDurabilityMode(*durabilityMode)
	if err != nil {
		slog.Error("Invalid durability mode", "error", err)
		os.Exit(1)
	}

	slog.Info("Starting "+branding.ProductName()+" broker", "node_id", *nodeID, "addr", *addr, "health_addr", *healthAddr)

	srv := server.NewServer(*addr, *dataDir, *maxSegSize, *indexInterval)
	srv.SetHealthAddr(*healthAddr)
	srv.SetShutdownTimeout(config.LifecycleShutdownTimeout())
	srv.SetDurability(blog.DurabilityConfig{
		Mode:               mode,
		FlushInterval:      *flushInterval,
		GroupCommitRecords: *groupCommitRecords,
	})

	if err := srv.Start(); err != nil {
		slog.Error("Failed to start server", "error", err)
		os.Exit(1)
	}

	coord, err := server.NewCoordinator(*nodeID, srv.Addr(), *redisURL, srv)
	if err != nil {
		slog.Error("Failed to initialize coordinator", "error", err)
		srv.Stop()
		os.Exit(1)
	}

	srv.SetCoordinator(coord)
	coord.Start()

	slog.Info(branding.ProductName() + " broker running")
	sig := lifecycle.WaitSignal()
	slog.Info("received shutdown signal", "signal", sig.String(), "node_id", *nodeID)

	slog.Info("Shutting down " + branding.ProductName() + " broker...")
	srv.Stop()
	coord.Stop()
	slog.Info("Shutdown complete.")
}
