package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"ad-event-processor/internal/edge"
	"ad-event-processor/pkg/lifecycle"
	"ad-event-processor/pkg/netaddr"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/ringbuf"
	"github.com/cilium/ebpf/rlimit"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
)

func main() {
	syncInterval := edge.EnvDuration("SYNC_INTERVAL", 5*time.Second)
	statsInterval := edge.EnvDuration("STATS_INTERVAL", 2*time.Second)
	violationInterval := edge.EnvDuration("VIOLATION_POLL_INTERVAL", 250*time.Millisecond)
	fingerprintInterval := edge.EnvDuration("FINGERPRINT_POLL_INTERVAL", 500*time.Millisecond)
	autobanTTL := edge.EnvDuration("AUTOBAN_TTL", 5*time.Minute)
	metricsPort := edge.EnvOr("METRICS_PORT", "9090")
	paths := edge.ResolvePinnedMapPaths()
	blocklistPath := paths.Blocklist
	blocklistV6Path := paths.BlocklistV6
	blocklistHostV4Path := paths.BlocklistHostV4
	blocklistHostV6Path := paths.BlocklistHostV6
	allowlistPath := paths.Allowlist
	allowlistV6Path := paths.AllowlistV6
	statsPath := paths.Stats
	violationsPath := paths.Violations
	fingerprintsPath := paths.Fingerprints
	redisAddr := edge.FirstRedisAddr()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	if redisAddr == "" {
		slog.Error("REDIS_ADDRS or REDIS_HOST/REDIS_PORT must be set")
		os.Exit(1)
	}

	if err := rlimit.RemoveMemlock(); err != nil {
		slog.Error("rlimit remove memlock", "error", err)
		os.Exit(1)
	}

	denyMap, err := edge.LoadPinnedBlocklistMap(blocklistPath)
	if err != nil {
		slog.Error("open pinned blocklist map", "path", blocklistPath, "error", err)
		os.Exit(1)
	}
	defer func() { _ = denyMap.Close() }()

	denyV6Map, err := edge.LoadPinnedBlocklistV6Map(blocklistV6Path)
	if err != nil {
		slog.Error("open pinned blocklist v6 map", "path", blocklistV6Path, "error", err)
		os.Exit(1)
	}
	defer func() { _ = denyV6Map.Close() }()

	denyHostV4Map, err := edge.LoadPinnedBlocklistHostV4Map(blocklistHostV4Path)
	if err != nil {
		slog.Error("open pinned blocklist host v4 map", "path", blocklistHostV4Path, "error", err)
		os.Exit(1)
	}
	defer func() { _ = denyHostV4Map.Close() }()

	denyHostV6Map, err := edge.LoadPinnedBlocklistHostV6Map(blocklistHostV6Path)
	if err != nil {
		slog.Error("open pinned blocklist host v6 map", "path", blocklistHostV6Path, "error", err)
		os.Exit(1)
	}
	defer func() { _ = denyHostV6Map.Close() }()

	denyMaps := edge.BlocklistMapsFromPinned(denyMap, denyV6Map, denyHostV4Map, denyHostV6Map)

	allowMap, err := edge.LoadPinnedAllowlistMap(allowlistPath)
	if err != nil {
		slog.Error("open pinned allowlist map", "path", allowlistPath, "error", err)
		os.Exit(1)
	}
	defer func() { _ = allowMap.Close() }()

	allowV6Map, err := edge.LoadPinnedAllowlistV6Map(allowlistV6Path)
	if err != nil {
		slog.Error("open pinned allowlist v6 map", "path", allowlistV6Path, "error", err)
		os.Exit(1)
	}
	defer func() { _ = allowV6Map.Close() }()

	statsMap, err := edge.LoadPinnedStatsMap(statsPath)
	if err != nil {
		slog.Warn("open pinned stats map; xdp metrics disabled", "path", statsPath, "error", err)
	} else {
		defer func() { _ = statsMap.Close() }()
	}

	var violationReader *ringbuf.Reader
	violationsMap, err := edge.LoadPinnedViolationsMap(violationsPath)
	if err != nil {
		slog.Warn("open pinned violations ringbuf; autoban disabled", "path", violationsPath, "error", err)
	} else {
		defer func() { _ = violationsMap.Close() }()
		violationReader, err = ringbuf.NewReader(violationsMap)
		if err != nil {
			slog.Warn("create violations ringbuf reader", "error", err)
		} else {
			defer func() { _ = violationReader.Close() }()
		}
	}

	var fingerprintReader *ringbuf.Reader
	fingerprintsMap, err := edge.LoadPinnedFingerprintsMap(fingerprintsPath)
	if err != nil {
		slog.Warn("open pinned fingerprints ringbuf; ivt staging disabled", "path", fingerprintsPath, "error", err)
	} else {
		defer func() { _ = fingerprintsMap.Close() }()
		fingerprintReader, err = ringbuf.NewReader(fingerprintsMap)
		if err != nil {
			slog.Warn("create fingerprints ringbuf reader", "error", err)
		} else {
			defer func() { _ = fingerprintReader.Close() }()
		}
	}

	redisClient := redis.NewClient(netaddr.RedisClientOptions(redisAddr, os.Getenv("REDIS_PASS")))
	defer func() { _ = redisClient.Close() }()

	ctx, cancel := lifecycle.NotifyContext(context.Background())
	defer cancel()

	go serveMetrics(ctx, metricsPort)

	denyStore := edge.NewBlocklistStore()
	allowStore := edge.NewAllowlistStore()
	var denySyncState edge.BlocklistSyncState
	var lastStats []uint64

	violationHandler := edge.NewViolationHandler(func(evt edge.ViolationEvent) error {
		ip := edge.HostIPv4(evt.SrcIP)
		if err := edge.RecordAutoBan(ctx, redisClient, ip, autobanTTL); err != nil {
			return err
		}
		slog.Info("xdp autoban recorded",
			"ip", ip,
			"reason", edge.ViolationReasonLabel(evt.Reason),
			"ttl", autobanTTL.String(),
		)
		return nil
	})

	fingerprintHandler := edge.NewFingerprintHandler(func(evt edge.FingerprintEvent) error {
		return edge.Record(ctx, redisClient, edge.Entry{
			IP:      edge.HostIPv4(evt.SrcIP),
			TCPHash: evt.TCPHash,
			TTL:     evt.TTL,
			Window:  evt.Window,
			MSS:     evt.MSS,
			SeenAt:  time.Now().UTC(),
		})
	})

	if edge.EbpfEdgeLicensed(ctx, redisClient) {
		if err := runSync(ctx, redisClient, denyMaps, allowMap, allowV6Map, denyStore, allowStore, &denySyncState); err != nil {
			slog.Warn("initial edge bpf sync failed", "error", err)
		}
	} else {
		slog.Warn("ebpf_xdp_edge module not licensed; edge-bpf-sync idle (maps pinned)")
	}

	if statsMap != nil {
		lastStats = exportStats(ctx, redisClient, statsMap, lastStats)
	}

	syncTicker := time.NewTicker(syncInterval)
	defer syncTicker.Stop()

	statsTicker := time.NewTicker(statsInterval)
	defer statsTicker.Stop()

	violationTicker := time.NewTicker(violationInterval)
	defer violationTicker.Stop()

	fingerprintTicker := time.NewTicker(fingerprintInterval)
	defer fingerprintTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("edge-bpf-sync stopped")
			return
		case <-violationTicker.C:
			if violationReader == nil || !edge.EbpfEdgeLicensed(ctx, redisClient) {
				continue
			}
			n, err := violationHandler.Drain(violationReader, violationInterval)
			if err != nil {
				slog.Warn("violation ringbuf drain failed", "error", err)
				continue
			}
			if n > 0 {
				if err := runSync(ctx, redisClient, denyMaps, allowMap, allowV6Map, denyStore, allowStore, &denySyncState); err != nil {
					slog.Warn("post-violation bpf sync failed", "error", err)
				}
			}
		case <-fingerprintTicker.C:
			if fingerprintReader == nil || !edge.EbpfEdgeLicensed(ctx, redisClient) {
				continue
			}
			if _, err := fingerprintHandler.Drain(fingerprintReader, fingerprintInterval); err != nil {
				slog.Warn("fingerprint ringbuf drain failed", "error", err)
			}
		case <-statsTicker.C:
			if statsMap != nil {
				lastStats = exportStats(ctx, redisClient, statsMap, lastStats)
			}
		case <-syncTicker.C:
			if !edge.EbpfEdgeLicensed(ctx, redisClient) {
				continue
			}
			if err := runSync(ctx, redisClient, denyMaps, allowMap, allowV6Map, denyStore, allowStore, &denySyncState); err != nil {
				slog.Warn("edge bpf sync failed", "error", err)
			}
		}
	}
}

func serveMetrics(ctx context.Context, port string) {
	mux := http.NewServeMux()
	mux.Handle("GET /metrics", promhttp.Handler())
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}
	lifecycle.ApplySidecarHTTPServerTimeouts(srv)
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()
	slog.Info("edge-bpf-sync metrics listening", "port", port)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("metrics server failed", "error", err)
	}
}

func runSync(ctx context.Context, redisClient *redis.Client, denyMaps edge.BlocklistMaps, allowMap, allowV6Map *ebpf.Map, denyStore *edge.BlocklistStore, allowStore *edge.AllowlistStore, denyState *edge.BlocklistSyncState) error {
	denyAdded, denyRemoved, err := edge.SyncBlocklistIncremental(ctx, redisClient, denyMaps, denyStore, denyState)
	if err != nil {
		return err
	}
	allowAdded, allowRemoved, err := edge.SyncAllowlistFromRedis(ctx, redisClient, allowMap, allowV6Map, allowStore)
	if err != nil {
		return err
	}
	edge.RecordBlocklistMapMetrics(denyMaps, denyStore)
	edge.RecordBlocklistChangelogLagSeconds(ctx, redisClient, denyState)
	slog.Info("edge bpf synced",
		"deny_entries", denyStore.Len(),
		"deny_added", denyAdded,
		"deny_removed", denyRemoved,
		"allow_entries", allowStore.Len(),
		"allow_added", allowAdded,
		"allow_removed", allowRemoved,
	)
	return nil
}

func exportStats(ctx context.Context, redisClient *redis.Client, statsMap *ebpf.Map, last []uint64) []uint64 {
	last = edge.ExportStatsToPrometheus(statsMap, last)
	totals, err := edge.AggregateStats(statsMap)
	if err != nil {
		return last
	}
	snap := edge.BuildSnapshot(totals)
	snap.UpdatedAt = time.Now().UTC()
	if err := edge.WriteRedis(ctx, redisClient, snap); err != nil {
		slog.Warn("write xdp stats snapshot", "error", err)
	}
	return last
}
