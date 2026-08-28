package opsadmin

import (
	"context"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/database"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type ReaderDeps struct {
	Pool                     *pgxpool.Pool
	RedisShards              []redis.UniversalClient
	Config                   *config.Config
	GetShardHealth           func(ctx context.Context) (ShardHealthReport, error)
	ListReconRuns            func(ctx context.Context, service string, limit, offset int32) ([]ReconRunDTO, int64, error)
	BuildStackHealthSnapshot func(ctx context.Context) (StackHealthSnapshot, error)
	ClickHouseQuery          *database.ClickHouseQuery
}

type Reader struct {
	deps ReaderDeps
}

func NewReader(deps ReaderDeps) *Reader {
	return &Reader{deps: deps}
}

func (r *Reader) pool() *pgxpool.Pool {
	if r == nil {
		return nil
	}
	return r.deps.Pool
}

func (r *Reader) cfg() *config.Config {
	if r == nil {
		return nil
	}
	return r.deps.Config
}

func (r *Reader) redisShards() []redis.UniversalClient {
	if r == nil {
		return nil
	}
	return r.deps.RedisShards
}
