package ingestion

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"time"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/metrics"

	"github.com/google/uuid"
)

type ReconciliationWorker struct {
	postgresConn         PostgresConn
	clickhouseConn ClickHouseConn
	repo           domain.CampaignRepository
	driftLimit     float64
	lag            time.Duration
	interval       time.Duration
}

func NewReconciliationWorker(
	postgresConn PostgresConn,
	clickhouseConn ClickHouseConn,
	repo domain.CampaignRepository,
	driftLimit float64,
	lag time.Duration,
	interval time.Duration,
) *ReconciliationWorker {
	return &ReconciliationWorker{
		postgresConn:         postgresConn,
		clickhouseConn: clickhouseConn,
		repo:           repo,
		driftLimit:     driftLimit,
		lag:            lag,
		interval:       interval,
	}
}

func (rw *ReconciliationWorker) Reconcile(ctx context.Context) error {
	campaigns, err := rw.repo.ListActive(ctx)
	if err != nil {
		return fmt.Errorf("reconciliation failed to list active campaigns: %w", err)
	}

	if len(campaigns) == 0 {
		return nil
	}

	until := time.Now().Add(-rw.lag)
	clickhouseSpends, err := rw.clickhouseConn.QueryAggregatedSpend(ctx, until)
	if err != nil {
		return fmt.Errorf("reconciliation failed to query ClickHouse aggregates: %w", err)
	}

	campaignIDs := make([]uuid.UUID, len(campaigns))
	for i, c := range campaigns {
		campaignIDs[i] = c.ID
	}
	postgresSpends, err := rw.postgresConn.GetCampaignSpends(ctx, campaignIDs)
	if err != nil {
		return fmt.Errorf("reconciliation failed to batch load Postgres spends: %w", err)
	}

	for _, c := range campaigns {
		postgresSpend := postgresSpends[c.ID]

		clickhouseSpend := clickhouseSpends[c.ID]

		var drift float64
		if postgresSpend > 0 {
			drift = math.Abs(float64(postgresSpend-clickhouseSpend)) / float64(postgresSpend)
		} else if clickhouseSpend > 0 {
			drift = 1.0
		}

		metrics.DataDriftRatio.WithLabelValues(c.ID.String()).Set(drift)

		if drift > rw.driftLimit {
			slog.Warn("Reconciliation: CRITICAL DATA DRIFT DETECTED",
				"campaign_id", c.ID,
				"postgres_spend", postgresSpend,
				"clickhouse_spend", clickhouseSpend,
				"drift_ratio", drift,
				"limit", rw.driftLimit,
			)
		} else {
			slog.Info("Reconciliation: campaign balances within normal drift limits",
				"campaign_id", c.ID,
				"postgres_spend", postgresSpend,
				"clickhouse_spend", clickhouseSpend,
				"drift_ratio", drift,
			)
		}
	}

	return nil
}

func (rw *ReconciliationWorker) Start(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(rw.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := rw.Reconcile(ctx); err != nil {
					slog.Error("Reconciliation: loop execution error", "error", err)
				}
			}
		}
	}()
}
