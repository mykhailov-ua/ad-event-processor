package controlplane

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"math"
	"time"

	"espx/internal/billing"
	"espx/internal/domain"
	db "espx/internal/domain/db"
	"espx/internal/metrics"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	redis "github.com/redis/go-redis/v9"
)

const (
	hyg30ReconDriftThresholdMicro = 1000
	hyg30CHStatsTolerancePct      = 0.0001
	hyg30LedgerSampleSize         = 100
)

func (w *ReconWorker) runHYG30Audits(ctx context.Context) {
	if w == nil || w.svc == nil {
		return
	}
	start := time.Now()
	pool := w.svc.settlePool()
	if err := w.svc.withPgLow(ctx, func(runCtx context.Context) error {
		w.auditRedisPGLedger(runCtx, pool)
		w.auditPGCHStats(runCtx)
		w.auditLedgerInvariantSample(runCtx, pool)
		return nil
	}); err != nil && !errors.Is(err, ErrMgmtPgGateRejected) {
		slog.Error("hyg30 recon audits failed", "error", err)
	}
	slog.Debug("hyg30 recon audits completed", "duration_ms", time.Since(start).Milliseconds())
}

func (w *ReconWorker) auditRedisPGLedger(ctx context.Context, pool *pgxpool.Pool) {
	if pool == nil {
		return
	}
	rows, err := pool.Query(ctx, `
		SELECT c.id, c.customer_id, c.current_spend,
		       COALESCE((SELECT SUM(CASE WHEN amount < 0 THEN -amount ELSE 0 END) FROM balance_ledger bl WHERE bl.campaign_id = c.id), 0)::bigint AS ledger_spend
		FROM campaigns c
		WHERE c.status IN ('ACTIVE', 'PAUSED', 'EXHAUSTED')
		ORDER BY c.updated_at DESC
		LIMIT 50`)
	if err != nil {
		slog.Error("hyg30 audit A query failed", "error", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var campID, customerID uuid.UUID
		var pgSpend, ledgerSpend int64
		if err := rows.Scan(&campID, &customerID, &pgSpend, &ledgerSpend); err != nil {
			continue
		}
		rdb := w.svc.getRDB(campID)
		if rdb == nil {
			continue
		}
		syncKey := domain.CampaignSyncKey(campID)
		redisSpend, err := rdb.Get(ctx, syncKey).Int64()
		if err != nil && !errors.Is(err, redis.Nil) {
			continue
		}
		drift := redisSpend - ledgerSpend
		if drift == 0 {
			metrics.ReconDriftMicro.DeleteLabelValues(campID.String())
			continue
		}
		absDrift := drift
		if absDrift < 0 {
			absDrift = -absDrift
		}
		metrics.ReconDriftMicro.WithLabelValues(campID.String()).Set(float64(absDrift))
		if absDrift <= hyg30ReconDriftThresholdMicro {
			continue
		}
		if w.svc.cfg != nil && w.svc.cfg.ReconForceRefillEnabled() {
			if err := w.svc.forceRefillCampaignFromPG(ctx, campID, pgSpend); err != nil {
				slog.Error("force refill from pg failed", "campaign_id", campID, "error", err)
			} else {
				slog.Info("force refill from pg applied", "campaign_id", campID, "pg_spend", pgSpend)
			}
		}
	}
}

func (w *ReconWorker) auditPGCHStats(ctx context.Context) {
	ch := w.svc.CHQuery()
	if ch == nil {
		return
	}
	rows, err := w.svc.GetPool().Query(ctx, `
		SELECT campaign_id, date, impressions_count + clicks_count + conversions_count AS pg_total
		FROM campaign_stats
		WHERE date >= CURRENT_DATE - INTERVAL '1 day'
		ORDER BY date DESC
		LIMIT 20`)
	if err != nil {
		slog.Error("hyg30 audit B pg query failed", "error", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var campID uuid.UUID
		var day time.Time
		var pgTotal int64
		if err := rows.Scan(&campID, &day, &pgTotal); err != nil {
			continue
		}
		if pgTotal == 0 {
			continue
		}
		var chTotal uint64
		chRows, err := ch.Query(ctx, `
			SELECT count() FROM ad_event_processor.events_analytics
			WHERE campaign_id = ? AND toDate(timestamp) = ?`, campID, day)
		if err != nil {
			continue
		}
		if chRows.Next() {
			_ = chRows.Scan(&chTotal)
		}
		chRows.Close()
		if chTotal == 0 {
			continue
		}
		diff := math.Abs(float64(int64(chTotal)-pgTotal)) / float64(pgTotal)
		if diff > hyg30CHStatsTolerancePct {
			slog.Warn("campaign stats stale vs clickhouse",
				"campaign_id", campID,
				"date", day.Format("2006-01-02"),
				"pg_total", pgTotal,
				"ch_total", chTotal,
				"diff_pct", diff,
			)
		}
	}
}

func (w *ReconWorker) auditLedgerInvariantSample(ctx context.Context, pool *pgxpool.Pool) {
	if pool == nil {
		return
	}
	rows, err := pool.Query(ctx, `SELECT id FROM customers ORDER BY RANDOM() LIMIT $1`, hyg30LedgerSampleSize)
	if err != nil {
		slog.Error("hyg30 audit C sample failed", "error", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var customerID uuid.UUID
		if err := rows.Scan(&customerID); err != nil {
			continue
		}
		if err := billing.CheckLedgerBalanceInvariant(ctx, pool, customerID); err != nil {
			slog.Error("ledger invariant failed for customer", "customer_id", customerID, "error", err)
			w.enqueueForcePauseCustomer(ctx, customerID, err.Error())
		}
	}
}

func (w *ReconWorker) enqueueForcePauseCustomer(ctx context.Context, customerID uuid.UUID, reason string) {
	if w.svc == nil || w.svc.GetPool() == nil {
		return
	}
	camps, err := w.svc.GetPool().Query(ctx, `
		SELECT id FROM campaigns WHERE customer_id = $1 AND status = 'ACTIVE'`, customerID)
	if err != nil {
		return
	}
	defer camps.Close()

	q := db.New(w.svc.GetPool())
	for camps.Next() {
		var campID uuid.UUID
		if err := camps.Scan(&campID); err != nil {
			continue
		}
		payload, _ := json.Marshal(map[string]string{
			"campaign_id": campID.String(),
			"reason":      "FORCE_PAUSE: " + reason,
		})
		_, err = q.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
			EventType: "PAUSE_CAMPAIGN",
			Payload:   payload,
		})
		if err != nil {
			slog.Error("failed to enqueue FORCE_PAUSE", "campaign_id", campID, "error", err)
		}
	}
}

func (s *Service) forceRefillCampaignFromPG(ctx context.Context, campaignID uuid.UUID, currentSpend int64) error {
	if s == nil {
		return errors.New("service nil")
	}
	var budgetLimit int64
	err := s.GetPool().QueryRow(ctx, `SELECT budget_limit FROM campaigns WHERE id = $1`, domain.ToUUID(campaignID)).Scan(&budgetLimit)
	if err != nil {
		return err
	}
	remaining := budgetLimit - currentSpend
	if remaining < 0 {
		remaining = 0
	}
	rdb := s.getRDB(campaignID)
	if rdb == nil {
		return errors.New("no redis shard")
	}
	key := domain.BudgetCampaignKey(campaignID)
	return rdb.Set(ctx, key, remaining, 24*time.Hour).Err()
}

func (s *Service) settlePool() *pgxpool.Pool {
	if s == nil {
		return nil
	}
	if s.settlePoolField != nil {
		return s.settlePoolField
	}
	return s.pool
}

func (s *Service) SetSettlePool(pool *pgxpool.Pool) {
	if s == nil {
		return
	}
	s.settlePoolField = pool
}
