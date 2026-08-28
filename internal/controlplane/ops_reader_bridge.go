package controlplane

import (
	"context"

	"ad-event-processor/internal/database"
	"ad-event-processor/internal/opsadmin"
)

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
