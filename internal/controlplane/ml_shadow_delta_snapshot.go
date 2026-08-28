package controlplane

import (
	"context"
	"time"

	"ad-event-processor/internal/reports"
)

const mlShadowDeltaSnapshotLookback = 7 * 24 * time.Hour

func (s *Service) refreshMLShadowDeltaSnapshot(ctx context.Context, now time.Time) error {
	if s == nil || s.clickhouseQuery == nil || s.GetPool() == nil {
		return nil
	}
	to := now.UTC()
	from := to.Add(-mlShadowDeltaSnapshotLookback)
	queryCtx, cancel := context.WithTimeout(ctx, reportClickHouseQueryTimeout)
	defer cancel()
	rows, _, err := reports.QueryMLShadowDeltaRows(queryCtx, s.clickhouseQuery, from, to, 10_000, 0)
	if err != nil {
		return err
	}
	return reports.UpsertMLShadowDeltaSnapshot(ctx, s.GetPool(), reports.MLShadowDeltaSnapshot{
		RangeFrom:   from,
		RangeTo:     to,
		GeneratedAt: now.UTC(),
		Rows:        rows,
	})
}
