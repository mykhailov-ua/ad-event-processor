package reconciliation

import (
	"context"

	"ad-event-processor/internal/reports"
)

// clickhouseQueryContext wraps reports.ClickHouseQueryContext (10s cold-path deadline for HYG30 CH audits).
func clickhouseQueryContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return reports.ClickHouseQueryContext(ctx)
}
