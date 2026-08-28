package fraudadmin

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type StaleEpochsHost interface {
	StaleEpochsRedisShards() []redis.UniversalClient
	StaleEpochsPool() *pgxpool.Pool
	StaleEpochsUpdateSettings(ctx context.Context, settings map[string]string) error
}
