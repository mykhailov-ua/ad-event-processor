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

	"github.com/bidshard/ad-event-processor/internal/domain"
	"github.com/bidshard/ad-event-processor/internal/metrics"

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
	if f == nil || f.script == nil || f.fastScript == nil {
		return fmt.Errorf("unified filter scripts are nil")
	}
	for i, rdb := range f.rdbs {
		if rdb == nil {
			continue
		}
		if err := f.preloadScriptsShard(ctx, i, rdb); err != nil {
			return err
		}
	}
	return f.openFilterEvalPins(ctx)
}

func (f *UnifiedFilter) preloadScriptsShard(ctx context.Context, shard int, rdb redis.UniversalClient) error {
	if f == nil || f.script == nil || f.fastScript == nil {
		return fmt.Errorf("unified filter scripts are nil")
	}
	if rdb == nil {
		return fmt.Errorf("preload filter scripts shard %d: redis client is nil", shard)
	}
	shardLabel := strconv.Itoa(shard)
	if err := f.script.Load(ctx, rdb).Err(); err != nil {
		metrics.RedisLuaScriptLoaded.WithLabelValues(shardLabel).Set(0)
		metrics.RedisLuaFastScriptLoaded.WithLabelValues(shardLabel).Set(0)
		return fmt.Errorf("preload filter full script shard %d: %w", shard, err)
	}
	if err := f.fastScript.Load(ctx, rdb).Err(); err != nil {
		metrics.RedisLuaScriptLoaded.WithLabelValues(shardLabel).Set(0)
		metrics.RedisLuaFastScriptLoaded.WithLabelValues(shardLabel).Set(0)
		return fmt.Errorf("preload budget fast script shard %d: %w", shard, err)
	}
	metrics.RedisLuaScriptLoaded.WithLabelValues(shardLabel).Set(1)
	metrics.RedisLuaFastScriptLoaded.WithLabelValues(shardLabel).Set(1)
	return nil
}

// AttachReconnectPreload registers DialHooks so new pooled connections SCRIPT LOAD
// after Redis failover/reconnect (not only on NOSCRIPT eval errors).
func (f *UnifiedFilter) AttachReconnectPreload() {
	if f == nil {
		return
	}
	for i, rdb := range f.rdbs {
		if rdb == nil {
			continue
		}
		rdb.AddHook(newRedisShardPreloadHook(f, i))
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
			h.schedulePreload()
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

func (h *redisShardPreloadHook) schedulePreload() {
	h.mu.Lock()
	if time.Since(h.last) < time.Second {
		h.mu.Unlock()
		return
	}
	h.last = time.Now()
	filter := h.filter
	shard := h.shard
	h.mu.Unlock()

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if filter == nil || shard < 0 || shard >= len(filter.rdbs) {
			return
		}
		rdb := filter.rdbs[shard]
		if rdb == nil {
			return
		}
		if err := filter.preloadScriptsShard(ctx, shard, rdb); err != nil {
			slog.Warn("redis lua reconnect preload failed", "shard", shard, "error", err)
		}
	}()
}

func (f *UnifiedFilter) evalScript(ctx context.Context, rdb redis.UniversalClient, shard int, evt *domain.Event, keyArgs [unifiedFilterKeyCount]any, args []any) (int64, error) {
	res, err := f.evalShaPooled(ctx, rdb, shard, evt, f.scriptHashAny, keyArgs, args)
	if err != nil && isNoScriptErr(err) {
		incRedisLuaNoScript(f.luaNoScriptCounters, shard)
		slog.Warn("redis lua NOSCRIPT encountered", "shard", shard, "error", err)

		go func() {
			ctxPreheat, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			_ = f.PreloadScripts(ctxPreheat)
		}()

		if f.evalFallbackGate != nil {
			select {
			case f.evalFallbackGate <- struct{}{}:
				defer func() { <-f.evalFallbackGate }()
				return f.evalPooled(ctx, rdb, shard, evt, unifiedFilterLuaAny, keyArgs, args)
			default:
				slog.Warn("redis lua NOSCRIPT fallback concurrency limit exceeded", "shard", shard)
				return -1, fmt.Errorf("redis lua EVAL fallback concurrency limit exceeded")
			}
		}
		return f.evalPooled(ctx, rdb, shard, evt, unifiedFilterLuaAny, keyArgs, args)
	}
	return res, err
}
