package ingestion

import (
	"context"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bidshard/ad-event-processor/internal/metrics"

	"github.com/redis/go-redis/v9"
)

type RedisStreamTrimmerConfig struct {
	Rdbs         []redis.UniversalClient
	Streams      []string
	MaxLen       int
	TrimInterval time.Duration
}

type RedisStreamTrimmer struct {
	cfg    RedisStreamTrimmerConfig
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func NewRedisStreamTrimmer(cfg RedisStreamTrimmerConfig) *RedisStreamTrimmer {
	if cfg.MaxLen <= 0 {
		cfg.MaxLen = 10000
	}
	if cfg.TrimInterval <= 0 {
		cfg.TrimInterval = 10 * time.Second
	}
	return &RedisStreamTrimmer{
		cfg: cfg,
	}
}

func (t *RedisStreamTrimmer) Start(ctx context.Context) {
	if len(t.cfg.Rdbs) == 0 {
		return
	}
	runCtx, cancel := context.WithCancel(ctx)
	t.cancel = cancel

	t.wg.Add(1)
	go func() {
		defer t.wg.Done()
		// Perform initial trim on startup
		t.TrimOnce(runCtx)

		ticker := time.NewTicker(t.cfg.TrimInterval)
		defer ticker.Stop()

		for {
			select {
			case <-runCtx.Done():
				return
			case <-ticker.C:
				t.TrimOnce(runCtx)
			}
		}
	}()
	slog.Info("redis stream trimmer started",
		"max_len", t.cfg.MaxLen,
		"trim_interval", t.cfg.TrimInterval.String(),
		"shards", len(t.cfg.Rdbs),
	)
}

func (t *RedisStreamTrimmer) TrimOnce(ctx context.Context) {
	for i, rdb := range t.cfg.Rdbs {
		if rdb == nil {
			continue
		}
		shardLabel := strconv.Itoa(i)

		// 1. Trim configured streams
		for _, stream := range t.cfg.Streams {
			if stream == "" {
				continue
			}
			cmd := rdb.XTrimMaxLenApprox(ctx, stream, int64(t.cfg.MaxLen), 0)
			if err := cmd.Err(); err != nil && err != redis.Nil {
				slog.Debug("redis stream xtrim error", "shard", i, "stream", stream, "error", err)
			}
		}

		// 2. Query Redis memory info and update telemetry metric ad_redis_memory_used_bytes
		infoCmd := rdb.Info(ctx, "memory")
		if res, err := infoCmd.Result(); err == nil {
			if usedBytes := parseRedisUsedMemory(res); usedBytes >= 0 {
				metrics.RedisMemoryUsedBytes.WithLabelValues(shardLabel).Set(float64(usedBytes))
			}
		}
	}
}

func parseRedisUsedMemory(info string) int64 {
	info = strings.ReplaceAll(info, "\r", "")
	lines := strings.Split(info, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "used_memory:") {
			parts := strings.Split(line, ":")
			if len(parts) == 2 {
				if val, err := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64); err == nil {
					return val
				}
			}
		}
	}
	return -1
}

func (t *RedisStreamTrimmer) Close() {
	if t.cancel != nil {
		t.cancel()
	}
}

func (t *RedisStreamTrimmer) Wait() {
	t.wg.Wait()
}
