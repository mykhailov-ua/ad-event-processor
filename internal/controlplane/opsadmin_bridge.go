package controlplane

import (
	"context"
	"log/slog"

	"ad-event-processor/internal/database"
	"ad-event-processor/internal/opsadmin"
)

var _ opsadmin.OpsMetricScraperHost = (*Service)(nil)

func (s *Service) startOpsMetricScraper(ctx context.Context, scrapeURL string) {
	opsadmin.StartMetricScraper(s, ctx, scrapeURL)
}

func NewManagementOpsReader(svc *Service) opsadmin.ManagementOpsReader {
	if svc == nil {
		return nil
	}
	var clickhouseQuery *database.ClickHouseQuery
	if svc.ClickHouseQuery() != nil {
		clickhouseQuery = svc.ClickHouseQuery()
	}
	return opsadmin.NewReader(opsadmin.ReaderDeps{
		Pool:        svc.GetPool(),
		RedisShards: svc.redisShards,
		Config:      svc.cfg,
		GetShardHealth: func(ctx context.Context) (opsadmin.ShardHealthReport, error) {
			report, err := svc.GetShardHealth(ctx)
			return report, err
		},
		ListReconRuns: svc.ListReconRuns,
		BuildStackHealthSnapshot: func(ctx context.Context) (opsadmin.StackHealthSnapshot, error) {
			return svc.BuildStackHealthSnapshot(ctx)
		},
		ClickHouseQuery: clickhouseQuery,
	})
}

func newOpsReader(svc *Service) opsadmin.ManagementOpsReader {
	return NewManagementOpsReader(svc)
}

func (s *Service) StartFilterRejectRollupWorker(ctx context.Context, scrapeURL string) {
	if s == nil || s.GetPool() == nil || s.clickhouseQuery == nil {
		slog.Warn("filter reject rollup worker not started: postgres or clickhouse unavailable")
		return
	}
	w := opsadmin.NewFilterRejectRollupWorker(s.GetPool(), s.clickhouseQuery, scrapeURL)
	w.SetEdgeFetcher(func(ctx context.Context) (map[string]uint64, error) {
		panel, err := opsadmin.FetchEdgeMetrics(ctx)
		if err != nil {
			return nil, err
		}
		return panel.Blocked, nil
	})
	s.StartBackgroundWorker(func() {
		w.Start(ctx)
	})
}
