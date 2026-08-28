package reports

import (
	"context"
	"time"
)

const coldPathClickHouseQueryTimeout = 10 * time.Second

func ClickHouseQueryContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, coldPathClickHouseQueryTimeout)
}
