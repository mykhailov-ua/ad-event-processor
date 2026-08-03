package ingestion

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"espx/internal/domain"
	"espx/internal/metrics"

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
		shard := strconv.Itoa(i)
		if err := f.script.Load(ctx, rdb).Err(); err != nil {
			metrics.RedisLuaScriptLoaded.WithLabelValues(shard).Set(0)
			metrics.RedisLuaFastScriptLoaded.WithLabelValues(shard).Set(0)
			return fmt.Errorf("preload filter full script shard %d: %w", i, err)
		}
		if err := f.fastScript.Load(ctx, rdb).Err(); err != nil {
			metrics.RedisLuaScriptLoaded.WithLabelValues(shard).Set(0)
			metrics.RedisLuaFastScriptLoaded.WithLabelValues(shard).Set(0)
			return fmt.Errorf("preload budget fast script shard %d: %w", i, err)
		}
		metrics.RedisLuaScriptLoaded.WithLabelValues(shard).Set(1)
		metrics.RedisLuaFastScriptLoaded.WithLabelValues(shard).Set(1)
	}
	return f.openFilterEvalPins(ctx)
}

func (f *UnifiedFilter) evalScript(ctx context.Context, rdb redis.UniversalClient, shard int, evt *domain.Event, keyArgs [unifiedFilterKeyCount]any, args []any) (int64, error) {
	res, err := f.evalShaPooled(ctx, rdb, shard, evt, f.scriptHashAny, keyArgs, args)
	if err != nil && isNoScriptErr(err) {
		incRedisLuaNoScript(f.luaNoScriptCounters, shard)
		return f.evalPooled(ctx, rdb, shard, evt, unifiedFilterLuaAny, keyArgs, args)
	}
	return res, err
}
