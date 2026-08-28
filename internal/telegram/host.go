package telegram

import (
	"ad-event-processor/internal/database"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type Host interface {
	Pool() *pgxpool.Pool
	RedisClientForCampaign(campaignID uuid.UUID) redis.UniversalClient
	ClickHouseQuery() *database.ClickHouseQuery
	ClickHouseWrite() driver.Conn
}
