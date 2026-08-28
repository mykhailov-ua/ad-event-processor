package shardadmin

import (
	"ad-event-processor/internal/config"
	"ad-event-processor/internal/pgfailover"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type PostgresFailoverHost interface {
	Pool() *pgxpool.Pool
	SetPool(*pgxpool.Pool)
	RedisShards() []redis.UniversalClient
	ControlRedis() redis.UniversalClient
	FailoverConfig() *config.Config
	SetPgFencing(*pgfailover.FencingGate)
	PgFencing() *pgfailover.FencingGate
}
