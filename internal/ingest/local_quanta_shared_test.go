package ingest

import (
	"context"
	"sync/atomic"

	redis "github.com/redis/go-redis/v9"
)

type evalCountRedis struct {
	redis.UniversalClient
	evals atomic.Int64
}

func (c *evalCountRedis) EvalSha(ctx context.Context, sha1 string, keys []string, args ...any) *redis.Cmd {
	c.evals.Add(1)
	return c.UniversalClient.EvalSha(ctx, sha1, keys, args...)
}

func (c *evalCountRedis) Eval(ctx context.Context, script string, keys []string, args ...any) *redis.Cmd {
	c.evals.Add(1)
	return c.UniversalClient.Eval(ctx, script, keys, args...)
}
