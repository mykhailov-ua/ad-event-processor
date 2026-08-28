package shardadmin

import (
	"context"

	"ad-event-processor/internal/dedup"

	"github.com/jackc/pgx/v5/pgxpool"
)

func DedupAdapter(ctx context.Context, pool *pgxpool.Pool, regionCode uint8) *dedup.Adapter {
	if pool == nil {
		return nil
	}
	epoch := dedup.LoadRoutingEpoch(ctx, pool)
	return dedup.NewAdapter(pool, regionCode, epoch)
}
