package outbox

import (
	"context"
	"time"

	"ad-event-processor/internal/dedup"
	"ad-event-processor/internal/shardadmin"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

const regionRelayOutboxTimeout = 30 * time.Second

type RegionRelayHost interface {
	Host
	Pool() *pgxpool.Pool
	WithPostgresHigh(ctx context.Context, fn func(context.Context) error) error
	RegionRelayRegionCode() uint8
	MultiRegionCell() bool
	DedupAdapter(ctx context.Context) *dedup.Adapter
	RedisShards() []redis.UniversalClient
	OperationLeaseWorker() *shardadmin.OperationLeaseWorker
	NewOperationLeaseWorker() *shardadmin.OperationLeaseWorker
	SetNXOnAllShards(ctx context.Context, key, value string, ttl time.Duration) (bool, error)
	RelayDeliveryBookRequest(ctx context.Context, regionCode uint8, outboxEventID int64, eventType string, payload []byte, attempt int32) shardadmin.OperationLeaseBookRequest
}
