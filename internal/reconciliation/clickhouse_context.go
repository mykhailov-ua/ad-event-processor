package reconciliation

import (
	"context"

	"ad-event-processor/internal/reports"
)

func clickhouseQueryContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return reports.ClickHouseQueryContext(ctx)
}
