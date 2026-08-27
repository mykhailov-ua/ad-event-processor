package controlplane

import (
	"context"
	"time"
)

const coldPathClickHouseQueryTimeout = 10 * time.Second

func clickhouseQueryContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, coldPathClickHouseQueryTimeout)
}
