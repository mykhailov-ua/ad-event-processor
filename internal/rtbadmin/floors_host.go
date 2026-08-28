package rtbadmin

import (
	"context"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/database"
	db "ad-event-processor/internal/domain/db"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type FloorsHost interface {
	FloorsPool() *pgxpool.Pool
	FloorsConfig() *config.Config
	FloorsClickHouse() *database.ClickHouseQuery
	FloorsRedisShards() []redis.UniversalClient
	FloorsEnqueueRtbCatalogReload(ctx context.Context, q db.Querier, trigger string) error
}
