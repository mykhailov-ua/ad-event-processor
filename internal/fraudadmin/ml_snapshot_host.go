package fraudadmin

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type MLSnapshotHost interface {
	SnapshotPool() *pgxpool.Pool
	SnapshotRedisShards() []redis.UniversalClient
}
