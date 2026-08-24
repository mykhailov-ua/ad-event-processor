package ingestion

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/metrics"

	redis "github.com/redis/go-redis/v9"
)

func isNoScriptErr(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, redis.ErrNoScript) {
		return true
	}
	return strings.Contains(err.Error(), "NOSCRIPT")
}

func (f *UnifiedFilter) PreloadScripts(ctx context.Context) error {
	if f == nil || f.script == nil || f.fastScript == nil || f.rollbackScript == nil {
		return fmt.Errorf("unified filter scripts are nil")
	}
	for i, redisClient := range f.redisShards {
		if redisClient == nil {
			continue
		}
		if err := f.preloadScriptsShard(ctx, i, redisClient); err != nil {
			return err
		}
	}
	return f.openFilterEvalPins(ctx)
}

func (f *UnifiedFilter) preloadScriptsShard(ctx context.Context, shard int, redisClient redis.UniversalClient) error {
	if f == nil || f.script == nil || f.fastScript == nil || f.rollbackScript == nil {
		return fmt.Errorf("unified filter scripts are nil")
	}
	if redisClient == nil {
		return fmt.Errorf("preload filter scripts shard %d: redis client is nil", shard)
	}
	shardLabel := strconv.Itoa(shard)
	if err := f.script.Load(ctx, redisClient).Err(); err != nil {
		metrics.RedisLuaScriptLoaded.WithLabelValues(shardLabel).Set(0)
		metrics.RedisLuaFastScriptLoaded.WithLabelValues(shardLabel).Set(0)
		return fmt.Errorf("preload filter full script shard %d: %w", shard, err)
	}
	if err := f.fastScript.Load(ctx, redisClient).Err(); err != nil {
		metrics.RedisLuaScriptLoaded.WithLabelValues(shardLabel).Set(0)
		metrics.RedisLuaFastScriptLoaded.WithLabelValues(shardLabel).Set(0)
		return fmt.Errorf("preload budget fast script shard %d: %w", shard, err)
	}
	if err := f.rollbackScript.Load(ctx, redisClient).Err(); err != nil {
		return fmt.Errorf("preload budget rollback script shard %d: %w", shard, err)
	}
	metrics.RedisLuaScriptLoaded.WithLabelValues(shardLabel).Set(1)
	metrics.RedisLuaFastScriptLoaded.WithLabelValues(shardLabel).Set(1)
	return nil
}

func (f *UnifiedFilter) AttachReconnectPreload() {
	if f == nil {
		return
	}
	for i, redisClient := range f.redisShards {
		if redisClient == nil {
			continue
		}
		redisClient.AddHook(newRedisShardPreloadHook(f, i))
	}
}

type redisShardPreloadHook struct {
	filter *UnifiedFilter
	shard  int
	mu     sync.Mutex
	last   time.Time
}

func newRedisShardPreloadHook(filter *UnifiedFilter, shard int) *redisShardPreloadHook {
	return &redisShardPreloadHook{filter: filter, shard: shard}
}

func (h *redisShardPreloadHook) DialHook(next redis.DialHook) redis.DialHook {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		conn, err := next(ctx, network, addr)
		if err == nil {
			h.schedulePreload(ctx)
		}
		return conn, err
	}
}

func (h *redisShardPreloadHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return next
}

func (h *redisShardPreloadHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return next
}

func (h *redisShardPreloadHook) schedulePreload(ctx context.Context) {
	if ctx == nil {
		return
	}
	h.mu.Lock()
	if time.Since(h.last) < time.Second {
		h.mu.Unlock()
		return
	}
	h.last = time.Now()
	filter := h.filter
	shard := h.shard
	h.mu.Unlock()

	go func(parent context.Context) {
		preloadCtx, cancel := context.WithTimeout(parent, 2*time.Second)
		defer cancel()
		if filter == nil || shard < 0 || shard >= len(filter.redisShards) {
			return
		}
		redisClient := filter.redisShards[shard]
		if redisClient == nil {
			return
		}
		if err := filter.preloadScriptsShard(preloadCtx, shard, redisClient); err != nil {
			slog.Warn("redis lua reconnect preload failed", "shard", shard, "error", err)
		}
	}(ctx)
}

func (f *UnifiedFilter) evalScript(ctx context.Context, redisClient redis.UniversalClient, shard int, evt *domain.Event, keyArgs [unifiedFilterKeyCount]any, args []any) (int64, error) {
	res, err := f.evalShaPooled(ctx, redisClient, shard, evt, f.scriptHashAny, keyArgs, args)
	if err != nil && isNoScriptErr(err) {
		incRedisLuaNoScript(f.luaNoScriptCounters, shard)
		slog.Warn("redis lua NOSCRIPT encountered", "shard", shard, "error", err)

		go func(parent context.Context) {
			ctxPreheat, cancel := context.WithTimeout(parent, 2*time.Second)
			defer cancel()
			_ = f.PreloadScripts(ctxPreheat)
		}(ctx)

		if f.evalFallbackGate != nil {
			select {
			case f.evalFallbackGate <- struct{}{}:
				defer func() { <-f.evalFallbackGate }()
				return f.evalPooled(ctx, redisClient, shard, evt, unifiedFilterLuaAny, keyArgs, args)
			default:
				slog.Warn("redis lua NOSCRIPT fallback concurrency limit exceeded", "shard", shard)
				return -1, fmt.Errorf("redis lua EVAL fallback concurrency limit exceeded")
			}
		}
		return f.evalPooled(ctx, redisClient, shard, evt, unifiedFilterLuaAny, keyArgs, args)
	}
	return res, err
}
