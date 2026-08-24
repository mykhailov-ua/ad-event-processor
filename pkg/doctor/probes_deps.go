package doctor

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/database"
	"ad-event-processor/pkg/iogate"
	"ad-event-processor/pkg/naming"

	"github.com/redis/go-redis/v9"
)

type RedisProbe struct {
	Deps ProbeDeps
}

func (RedisProbe) Name() string { return "redis" }

func (p RedisProbe) Run(ctx context.Context) Result {
	start := time.Now()
	if p.Deps.Config == nil {
		return Result{Name: "redis", Status: StatusSkip, Detail: "config not loaded", Latency: time.Since(start).Milliseconds()}
	}
	clients, err := p.redisClients(ctx)
	if err != nil {
		return Result{Name: "redis", Status: StatusFail, Detail: err.Error(), Latency: time.Since(start).Milliseconds()}
	}
	defer closeRedisShards(clients)

	var maxLatency time.Duration
	for i, rdb := range clients {
		shardStart := time.Now()
		if err := rdb.Ping(ctx).Err(); err != nil {
			return Result{
				Name:    "redis",
				Status:  StatusFail,
				Detail:  fmt.Sprintf("shard %d ping: %v", i, err),
				Latency: time.Since(start).Milliseconds(),
			}
		}
		if lat := time.Since(shardStart); lat > maxLatency {
			maxLatency = lat
		}
	}
	status := StatusPass
	if maxLatency > 10*time.Millisecond {
		status = StatusWarn
	}
	return Result{
		Name:    "redis",
		Status:  status,
		Detail:  fmt.Sprintf("%d shards ok; max ping %s", len(clients), maxLatency),
		Latency: time.Since(start).Milliseconds(),
	}
}

func (p RedisProbe) redisClients(ctx context.Context) ([]redis.UniversalClient, error) {
	if p.Deps.Redis != nil {
		return p.Deps.Redis(ctx)
	}
	clients, _, err := database.ConnectRedisShards(ctx, p.Deps.Config, database.RedisShardOptions{PoolSize: 2})
	return clients, err
}

func closeRedisShards(clients []redis.UniversalClient) {
	for _, c := range clients {
		if c != nil {
			_ = c.Close()
		}
	}
}

type ClickHouseProbe struct {
	Deps ProbeDeps
}

func (ClickHouseProbe) Name() string { return "clickhouse" }

func (p ClickHouseProbe) Run(ctx context.Context) Result {
	start := time.Now()
	if os.Getenv("CH_ENABLED") == "0" {
		return Result{Name: "clickhouse", Status: StatusSkip, Detail: "CH_ENABLED=0", Latency: time.Since(start).Milliseconds()}
	}
	if p.Deps.Config == nil || !p.Deps.Config.ClickHouseEnabled() {
		return Result{Name: "clickhouse", Status: StatusSkip, Detail: "clickhouse not configured", Latency: time.Since(start).Milliseconds()}
	}
	ping := p.Deps.CHPing
	if ping == nil {
		ping = defaultCHPing(p.Deps.Config)
	}
	if err := ping(ctx); err != nil {
		return Result{Name: "clickhouse", Status: StatusFail, Detail: err.Error(), Latency: time.Since(start).Milliseconds()}
	}
	latency := time.Since(start)
	status := StatusPass
	if latency > 500*time.Millisecond {
		status = StatusWarn
	}
	return Result{Name: "clickhouse", Status: status, Detail: latency.String(), Latency: latency.Milliseconds()}
}

func defaultCHPing(cfg *config.Config) func(context.Context) error {
	return func(ctx context.Context) error {
		conn, err := database.ConnectClickHouse(ctx, string(cfg.CHDSN))
		if err != nil {
			return err
		}
		defer func() { _ = conn.Close() }()
		return conn.Exec(ctx, "INSERT INTO FUNCTION Null('doctor_probe UInt8') VALUES (1)")
	}
}

type DiskProbe struct{}

func (DiskProbe) Name() string { return "disk" }

func (DiskProbe) Run(ctx context.Context) Result {
	start := time.Now()
	budget := iogate.DefaultConfig().DiskLatencyBudget
	path := filepath.Join(os.TempDir(), "ad-event-processor-doctor-disk-smoke")
	payload := make([]byte, 4096)
	for i := range payload {
		payload[i] = byte(i)
	}
	writeStart := time.Now()
	if err := os.WriteFile(path, payload, 0o600); err != nil {
		return Result{Name: "disk", Status: StatusFail, Detail: err.Error(), Latency: time.Since(start).Milliseconds()}
	}
	defer func() { _ = os.Remove(path) }()
	latency := time.Since(writeStart)
	status := StatusPass
	if latency > budget {
		status = StatusWarn
	}
	return Result{
		Name:    "disk",
		Status:  status,
		Detail:  fmt.Sprintf("write 4KiB in %s (budget %s)", latency, budget),
		Latency: time.Since(start).Milliseconds(),
	}
}

type TLSProbe struct {
	Deps ProbeDeps
}

func (TLSProbe) Name() string { return "tls" }

func (p TLSProbe) Run(ctx context.Context) Result {
	start := time.Now()
	if os.Getenv(naming.LegacyVendorEnvKey("PROFILE")) != "production" {
		return Result{Name: "tls", Status: StatusSkip, Detail: naming.LegacyVendorEnvKey("PROFILE") + "!=production", Latency: time.Since(start).Milliseconds()}
	}
	if p.Deps.Config == nil || string(p.Deps.Config.DBDSN) == "" {
		return Result{Name: "tls", Status: StatusFail, Detail: "DB_DSN not set", Latency: time.Since(start).Milliseconds()}
	}
	sslmode, err := dsnSSLMode(string(p.Deps.Config.DBDSN))
	if err != nil {
		return Result{Name: "tls", Status: StatusFail, Detail: err.Error(), Latency: time.Since(start).Milliseconds()}
	}
	if sslmode != "verify-full" {
		return Result{
			Name:    "tls",
			Status:  StatusFail,
			Detail:  fmt.Sprintf("DB_DSN sslmode=%q want verify-full", sslmode),
			Latency: time.Since(start).Milliseconds(),
		}
	}
	return Result{Name: "tls", Status: StatusPass, Detail: "postgres sslmode=verify-full", Latency: time.Since(start).Milliseconds()}
}

func dsnSSLMode(dsn string) (string, error) {
	u, err := url.Parse(dsn)
	if err != nil {
		return "", fmt.Errorf("parse DB_DSN: %w", err)
	}
	q := u.Query()
	mode := q.Get("sslmode")
	if mode == "" {
		return "disable", nil
	}
	return mode, nil
}
