package fraudadmin

import (
	"context"
	"time"

	"ad-event-processor/internal/reports"
)

const mlShadowDeltaSnapshotLookback = 7 * 24 * time.Hour

func RefreshMLShadowDeltaSnapshot(ctx context.Context, host MLShadowDeltaSnapshotHost, now time.Time) error {
	if host == nil || host.ClickHouseQuery() == nil || host.SnapshotPool() == nil {
		return nil
	}
	to := now.UTC()
	from := to.Add(-mlShadowDeltaSnapshotLookback)
	queryCtx, cancel := context.WithTimeout(ctx, reports.ReportClickHouseQueryTimeout())
	defer cancel()
	rows, _, err := reports.QueryMLShadowDeltaRows(queryCtx, host.ClickHouseQuery(), from, to, 10_000, 0)
	if err != nil {
		return err
	}
	return reports.UpsertMLShadowDeltaSnapshot(ctx, host.SnapshotPool(), reports.MLShadowDeltaSnapshot{
		RangeFrom:   from,
		RangeTo:     to,
		GeneratedAt: now.UTC(),
		Rows:        rows,
	})
}
