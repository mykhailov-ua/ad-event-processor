package controlplane

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"time"

	"espx/internal/billing"
	"espx/internal/config"
	"espx/internal/domain"
	db "espx/internal/domain/db"
	"espx/internal/metrics"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	redis "github.com/redis/go-redis/v9"
)

func parseReconciliationAdjustPayload(payload []byte) (ReconciliationAdjustPayload, error) {
	var p ReconciliationAdjustPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return p, err
	}
	if p.CampaignID == "" || p.CustomerID == "" {
		return p, fmt.Errorf("invalid reconciliation adjust payload")
	}
	if p.LedgerAmt == 0 && p.RedisDelta == 0 {
		return p, fmt.Errorf("empty reconciliation adjust")
	}
	return p, nil
}

func (w *OutboxWorker) ApplyReconciliationAdjust(ctx context.Context, eventID int64, payload []byte) error {
	p, err := parseReconciliationAdjustPayload(payload)
	if err != nil {
		return err
	}
	campID, err := uuid.Parse(p.CampaignID)
	if err != nil {
		return fmt.Errorf("invalid campaign id: %w", err)
	}
	customerID, err := uuid.Parse(p.CustomerID)
	if err != nil {
		return fmt.Errorf("invalid customer id: %w", err)
	}

	tx, err := w.svc.GetPool().Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	q := db.New(tx)
	_, err = q.CreateLedgerEntry(ctx, db.CreateLedgerEntryParams{
		CustomerID: pgUUID(customerID),
		CampaignID: domain.ToUUID(campID),
		Amount:     p.LedgerAmt,
		Type:       db.LedgerTypeRECONCILIATIONADJUST,
	})
	if err != nil {
		return err
	}

	spendDelta := -p.LedgerAmt
	if spendDelta != 0 {
		if err := q.UpdateCampaignSpend(ctx, db.UpdateCampaignSpendParams{
			ID:           domain.ToUUID(campID),
			CurrentSpend: spendDelta,
		}); err != nil {
			return err
		}
	}

	if p.RedisDelta != 0 {
		if int(p.ShardID) >= len(w.svc.rdbs) {
			return fmt.Errorf("invalid shard_id %d", p.ShardID)
		}
		rdb := w.svc.rdbs[p.ShardID]
		recon := NewReconService(w.svc)
		if err := recon.adjustRedisBudgetAtomically(ctx, rdb, campID, p.RedisDelta); err != nil {
			return err
		}
	}

	if p.RunID > 0 {
		_, err = tx.Exec(ctx, `
			UPDATE recon_discrepancies
			SET redis_adjusted = true
			WHERE run_id = $1 AND campaign_id = $2`,
			p.RunID, domain.ToUUID(campID),
		)
		if err != nil {
			return err
		}
	}

	adminID := uuid.MustParse(quotaRepairSystemAdmin)
	w.svc.AuditLog(ctx, q, adminID, "RECONCILIATION_ADJUST", "campaign",
		&campID, p, map[string]any{"outbox_event_id": eventID})

	if err := tx.Commit(ctx); err != nil {
		return err
	}
	metrics.ReconCorrectionsAppliedTotal.Inc()
	return nil
}

func pgUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: true}
}

type ReconService struct {
	mgmt *Service
}

func NewReconService(svc *Service) *ReconService {
	return &ReconService{mgmt: svc}
}

func (reconService *ReconService) ReconcileWindow(ctx context.Context, start, end time.Time) error {
	opCtx, cancel := workerContext(ctx, workerBatchTimeout)
	defer cancel()

	run, err := reconService.createRun(opCtx, start, end)
	if err != nil {
		slog.Error("failed to create recon run record", "error", err, "start", start, "end", end)
		metrics.ReconRunsTotal.WithLabelValues("failed").Inc()
		return err
	}

	ledgerRows, err := reconService.mgmt.GetPool().Query(opCtx, `
		SELECT campaign_id, COALESCE(SUM(CASE WHEN amount < 0 THEN -amount ELSE 0 END), 0)::bigint
		FROM balance_ledger
		WHERE created_at >= $1 AND created_at < $2
		  AND (type IN ('FEE', 'RECONCILIATION_ADJUST', 'REFUND'))
		GROUP BY campaign_id
	`, start, end)
	if err != nil {
		reconService.failRun(opCtx, run.ID, err)
		metrics.ReconRunsTotal.WithLabelValues("failed").Inc()
		return err
	}
	defer ledgerRows.Close()

	ledgerMap := make(map[uuid.UUID]int64)
	for ledgerRows.Next() {
		var cid uuid.UUID
		var spent int64
		if err := ledgerRows.Scan(&cid, &spent); err != nil {
			slog.Error("failed to scan ledger row in recon run", "run_id", run.ID, "error", err)
			continue
		}
		ledgerMap[cid] = spent
	}

	discrepancies := 0
	var totalDelta int64

	for campID, ledgerSpent := range ledgerMap {
		syncKey := domain.CampaignSyncKey(campID)
		rdb := reconService.mgmt.getRDB(campID)
		if rdb == nil {
			slog.Error("no redis shard for campaign in recon", "campaign_id", campID)
			metrics.ReconAdjustmentErrors.Inc()
			continue
		}

		syncVal, err := rdb.Get(opCtx, syncKey).Int64()
		if err != nil && !errors.Is(err, redis.Nil) {
			slog.Error("failed to fetch campaign sync budget from Redis in recon", "campaign_id", campID, "error", err)
			metrics.ReconAdjustmentErrors.Inc()
			reconService.failRun(opCtx, run.ID, err)
			metrics.ReconRunsTotal.WithLabelValues("failed").Inc()
			return err
		}

		delta := syncVal - ledgerSpent
		if delta == 0 {
			continue
		}

		var customerID pgtype.UUID
		err = reconService.mgmt.GetPool().QueryRow(opCtx, `SELECT customer_id FROM campaigns WHERE id = $1`, domain.ToUUID(campID)).Scan(&customerID)
		if err != nil {
			slog.Error("failed to resolve campaign customer in recon", "campaign_id", campID, "error", err)
			metrics.ReconAdjustmentErrors.Inc()
			continue
		}

		discrepancies++

		_, err = reconService.mgmt.GetPool().Exec(opCtx, `
			INSERT INTO recon_discrepancies (run_id, campaign_id, customer_id, expected_spend, actual_spend, delta, redis_adjusted)
			VALUES ($1, $2, $3, $4, $5, $6, false)
		`, run.ID, domain.ToUUID(campID), customerID, syncVal, ledgerSpent, delta)
		if err != nil {
			slog.Error("failed to record recon discrepancy to postgres", "run_id", run.ID, "campaign_id", campID, "error", err)
			metrics.ReconAdjustmentErrors.Inc()
			reconService.failRun(opCtx, run.ID, err)
			metrics.ReconRunsTotal.WithLabelValues("failed").Inc()
			return err
		}

		chunkMicro := reconService.autoAdjustChunkMicro()
		if abs(delta) > chunkMicro {
			slog.Warn("recon discrepancy exceeds auto-adjust chunk, leaving unresolved",
				"run_id", run.ID,
				"campaign_id", campID,
				"delta", delta,
				"chunk_micro", chunkMicro,
			)
			totalDelta += delta
			continue
		}

		shardID := int16(reconService.mgmt.sharder.GetShard(campID))
		custUUID, _ := uuid.FromBytes(customerID.Bytes[:])
		if err := reconService.enqueueReconciliationAdjust(opCtx, run.ID, campID, custUUID, shardID, -delta, delta, "hourly_window_recon"); err != nil {
			slog.Error("failed to enqueue recon adjustment", "campaign_id", campID, "delta", delta, "error", err)
			metrics.ReconAdjustmentErrors.Inc()
			continue
		}
		metrics.ReconCorrectionsTotal.Inc()
		totalDelta += delta
	}

	_, err = reconService.mgmt.GetPool().Exec(opCtx, `
		UPDATE recon_runs 
		SET status = 'COMPLETED', total_delta = $1, campaigns_checked = $2, discrepancies_found = $3, completed_at = NOW()
		WHERE id = $4
	`, totalDelta, len(ledgerMap), discrepancies, run.ID)
	if err != nil {
		slog.Error("failed to finalize recon run in postgres", "run_id", run.ID, "error", err)
		metrics.ReconRunsTotal.WithLabelValues("failed").Inc()
		return err
	}

	metrics.ReconRunsTotal.WithLabelValues("success").Inc()
	if discrepancies > 0 {
		metrics.ReconDiscrepanciesTotal.Add(float64(discrepancies))
	}
	metrics.ReconTotalDelta.Add(float64(abs(totalDelta)))

	slog.Info("reconciliation completed",
		"period", start.Format(time.RFC3339)+"-"+end.Format(time.RFC3339),
		"delta", totalDelta,
		"discrepancies", discrepancies,
	)
	if discrepancies > 0 && reconService.mgmt.alerter != nil {
		reconService.mgmt.alerter.AlertReconDiscrepancy(
			run.ID,
			discrepancies,
			totalDelta,
			start.Format(time.RFC3339)+"-"+end.Format(time.RFC3339),
		)
	}
	return nil
}

func (reconService *ReconService) autoAdjustChunkMicro() int64 {
	if reconService.mgmt.cfg != nil && reconService.mgmt.cfg.QuotaChunkSize > 0 {
		return reconService.mgmt.cfg.QuotaChunkSize
	}
	return 5_000_000
}

func (reconService *ReconService) AlertStaleUnresolvedDiscrepancies(ctx context.Context) {
	if reconService.mgmt.alerter == nil {
		return
	}
	pool := reconService.mgmt.GetPool()
	if pool == nil {
		return
	}

	rows, err := pool.Query(ctx, `
		SELECT d.run_id,
		       COUNT(*)::int,
		       COALESCE(SUM(ABS(d.delta)), 0)::bigint,
		       MIN(d.created_at) AS oldest,
		       r.period_start,
		       r.period_end
		FROM recon_discrepancies d
		JOIN recon_runs r ON r.id = d.run_id
		WHERE d.redis_adjusted = false
		  AND d.created_at < NOW() - INTERVAL '1 hour'
		GROUP BY d.run_id, r.period_start, r.period_end`)
	if err != nil {
		slog.Error("failed to query stale unresolved recon discrepancies", "error", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var runID int64
		var unresolved int
		var totalDelta int64
		var oldest time.Time
		var periodStart, periodEnd time.Time
		if err := rows.Scan(&runID, &unresolved, &totalDelta, &oldest, &periodStart, &periodEnd); err != nil {
			slog.Error("failed to scan stale recon discrepancy row", "error", err)
			continue
		}
		period := periodStart.Format(time.RFC3339) + "-" + periodEnd.Format(time.RFC3339)
		reconService.mgmt.alerter.AlertReconDiscrepancyUnresolved(runID, unresolved, totalDelta, period, oldest)
	}
}

func (reconService *ReconService) adjustRedisBudgetAtomically(ctx context.Context, rdb redis.UniversalClient, campID uuid.UUID, delta int64) error {
	script := `
		local key = KEYS[1]
		local delta = tonumber(ARGV[1])
		local newVal = redis.call("INCRBY", key, delta)
		if newVal <= 0 then
			redis.call("DEL", key)
			return 0
		end
		return newVal
	`
	_, err := rdb.Eval(ctx, script, []string{domain.CampaignSyncKey(campID)}, delta).Result()
	return err
}

func (reconService *ReconService) createRun(ctx context.Context, start, end time.Time) (struct{ ID int64 }, error) {
	var run struct{ ID int64 }
	err := reconService.mgmt.GetPool().QueryRow(ctx, `
		INSERT INTO recon_runs (period_start, period_end, status) VALUES ($1, $2, 'PENDING') RETURNING id
	`, start, end).Scan(&run.ID)
	return run, err
}

func (reconService *ReconService) enqueueReconciliationAdjust(
	ctx context.Context,
	runID int64,
	campID, customerID uuid.UUID,
	shardID int16,
	ledgerAmt, redisDelta int64,
	reason string,
) error {
	worker := &ReconWorker{svc: reconService.mgmt}
	return worker.enqueueReconciliationAdjust(ctx, runID, campID, customerID, shardID, ledgerAmt, redisDelta, reason)
}

func (reconService *ReconService) failRun(ctx context.Context, id int64, err error) {
	_, execErr := reconService.mgmt.GetPool().Exec(ctx, `UPDATE recon_runs SET status = 'FAILED' WHERE id = $1`, id)
	if execErr != nil {
		slog.Error("failed to mark recon run status as failed in postgres", "run_id", id, "error", execErr)
	}
	slog.Error("reconciliation run failed", "run_id", id, "error", err)
}

func abs(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}

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

const reconciliationAdjustEventType = "RECONCILIATION_ADJUST"

type BrokerPendingDeltaReader interface {
	PendingDeltaMicro(ctx context.Context, campaignID uuid.UUID) (int64, error)
}

type ReconciliationAdjustPayload struct {
	RunID      int64  `json:"run_id,omitempty"`
	CampaignID string `json:"campaign_id"`
	CustomerID string `json:"customer_id"`
	ShardID    int16  `json:"shard_id"`
	LedgerAmt  int64  `json:"ledger_amount_micro"`
	RedisDelta int64  `json:"redis_delta_micro"`
	Reason     string `json:"reason"`
}

type campaignBudgetPG struct {
	campaignID    uuid.UUID
	customerID    uuid.UUID
	budgetLimit   int64
	currentSpend  int64
	quotaReserved int64
	updatedAt     time.Time
}

func (w *ReconWorker) ReconcileBudgetSnapshot(ctx context.Context) {
	if w == nil || w.svc == nil || w.svc.GetPool() == nil {
		return
	}
	w.observeShardQuorum(ctx)

	reconSvc := NewReconService(w.svc)
	campaignIDs, err := w.collectDirtyCampaignIDs(ctx)
	if err != nil {
		slog.Error("budget snapshot recon: dirty set scan failed", "error", err)
		return
	}

	var checked, skipped, discrepancies int
	for _, campID := range campaignIDs {
		ok, disc, skip := w.reconcileCampaignSnapshot(ctx, reconSvc, campID)
		if skip {
			skipped++
			continue
		}
		if ok {
			checked++
		}
		if disc {
			discrepancies++
		}
	}

	if discrepancies > 0 {
		metrics.ReconDiscrepanciesTotal.Add(float64(discrepancies))
	}
	slog.Debug("budget snapshot recon completed",
		"checked", checked,
		"skipped", skipped,
		"discrepancies", discrepancies,
	)
}

func (w *ReconWorker) collectDirtyCampaignIDs(ctx context.Context) ([]uuid.UUID, error) {
	seen := make(map[uuid.UUID]struct{})
	for shardIdx, rdb := range w.svc.rdbs {
		if w.quorum != nil && w.quorum.DeadShardConfirmed(shardIdx) {
			continue
		}
		var cursor uint64
		for {
			keys, next, err := rdb.SScan(ctx, "budget:dirty_campaigns", cursor, "", 200).Result()
			if err != nil {
				return nil, err
			}
			for _, idStr := range keys {
				id, err := uuid.Parse(idStr)
				if err != nil {
					continue
				}
				seen[id] = struct{}{}
			}
			if next == 0 {
				break
			}
			cursor = next
		}
	}
	out := make([]uuid.UUID, 0, len(seen))
	for id := range seen {
		out = append(out, id)
	}
	return out, nil
}

func (w *ReconWorker) reconcileCampaignSnapshot(ctx context.Context, reconSvc *ReconService, campID uuid.UUID) (checked, discrepancy, skipped bool) {
	shardIdx := w.svc.sharder.GetShard(campID)
	if w.quorum != nil && w.quorum.DeadShardConfirmed(shardIdx) {
		return false, false, true
	}
	if shardIdx >= len(w.svc.rdbs) {
		return false, false, true
	}
	rdb := w.svc.rdbs[shardIdx]

	pg, err := w.loadCampaignBudgetPG(ctx, campID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, false, true
		}
		slog.Error("budget snapshot recon: load pg state failed", "campaign_id", campID, "error", err)
		return false, false, true
	}

	quotaMode := w.svc.cfg != nil && (w.svc.cfg.QuotaMode == "shadow" || w.svc.cfg.QuotaMode == "live")
	snap, err := domain.FetchBudgetReconSnapshot(ctx, rdb, campID, quotaMode)
	if err != nil {
		slog.Error("budget snapshot recon: redis snapshot failed", "campaign_id", campID, "error", err)
		return false, false, true
	}

	if snap.HasFence {
		return false, false, true
	}
	if snap.HasLock {
		return false, false, true
	}
	if w.shouldSkipSnapshotGrace(snap, pg.updatedAt) {
		return false, false, true
	}

	brokerPending := int64(0)
	if w.svc.brokerDeltas != nil {
		brokerPending, _ = w.svc.brokerDeltas.PendingDeltaMicro(ctx, campID)
	}

	pgRemaining := pg.budgetLimit - pg.currentSpend
	redisTotal := snap.RedisBudgetRemainingTotal(brokerPending)
	drift := pgRemaining - redisTotal
	tolerance := reconToleranceMicro(pg.budgetLimit)
	if abs(drift) <= tolerance {
		return true, false, false
	}

	metrics.ReconDriftMicro.WithLabelValues(campID.String()).Set(float64(abs(drift)))

	runID, err := reconSvc.createSnapshotRun(ctx)
	if err != nil {
		slog.Error("budget snapshot recon: create run failed", "error", err)
		return true, false, false
	}

	_, err = w.svc.GetPool().Exec(ctx, `
		INSERT INTO recon_discrepancies (run_id, campaign_id, customer_id, expected_spend, actual_spend, delta, redis_adjusted)
		VALUES ($1, $2, $3, $4, $5, $6, false)`,
		runID, domain.ToUUID(campID), pgtype.UUID{Bytes: pg.customerID, Valid: true},
		redisTotal, pgRemaining, drift,
	)
	if err != nil {
		slog.Error("budget snapshot recon: record discrepancy failed", "campaign_id", campID, "error", err)
		return true, false, false
	}

	chunk := reconSvc.autoAdjustChunkMicro()
	if abs(drift) > chunk {
		slog.Warn("budget snapshot drift exceeds auto-adjust chunk",
			"campaign_id", campID, "drift", drift, "chunk", chunk)
		return true, true, false
	}

	correction := drift
	if correction > chunk {
		correction = chunk
	} else if correction < -chunk {
		correction = -chunk
	}

	if err := w.enqueueReconciliationAdjust(ctx, runID, campID, pg.customerID, int16(shardIdx), -correction, correction, "budget_snapshot_invariant"); err != nil {
		slog.Error("budget snapshot recon: enqueue adjust failed", "campaign_id", campID, "error", err)
		metrics.ReconAdjustmentErrors.Inc()
		return true, true, false
	}
	metrics.ReconCorrectionsTotal.Inc()
	return true, true, false
}

func (w *ReconWorker) loadCampaignBudgetPG(ctx context.Context, campID uuid.UUID) (campaignBudgetPG, error) {
	var out campaignBudgetPG
	out.campaignID = campID
	err := w.svc.GetPool().QueryRow(ctx, `
		SELECT c.customer_id, c.budget_limit, c.current_spend, c.updated_at,
		       COALESCE(q.reserved_amount, 0)
		FROM campaigns c
		LEFT JOIN campaign_quotas q ON q.campaign_id = c.id
		WHERE c.id = $1`,
		domain.ToUUID(campID),
	).Scan(&out.customerID, &out.budgetLimit, &out.currentSpend, &out.updatedAt, &out.quotaReserved)
	return out, err
}

func (w *ReconWorker) shouldSkipSnapshotGrace(snap domain.BudgetReconSnapshot, lastPGUpdate time.Time) bool {
	if snap.Inflight <= 0 {
		return false
	}
	grace := reconGraceWindow(w.svc.cfg)
	return time.Since(lastPGUpdate) < grace
}

func reconGraceWindow(cfg *config.Config) time.Duration {
	if cfg == nil {
		return 15 * time.Second
	}
	ms := cfg.LedgerBatchFlushMs + cfg.BudgetSyncIntervalMs
	if ms <= 0 {
		return 15 * time.Second
	}
	return time.Duration(ms) * time.Millisecond
}

func reconToleranceMicro(budgetLimit int64) int64 {
	pct := int64(math.Max(1, float64(budgetLimit)*0.0001))
	if pct < 1 {
		return 1
	}
	return pct
}

func (reconService *ReconService) createSnapshotRun(ctx context.Context) (int64, error) {
	now := time.Now().UTC()
	var id int64
	err := reconService.mgmt.GetPool().QueryRow(ctx, `
		INSERT INTO recon_runs (period_start, period_end, status) VALUES ($1, $2, 'SNAPSHOT') RETURNING id`,
		now, now,
	).Scan(&id)
	return id, err
}

func (w *ReconWorker) enqueueReconciliationAdjust(
	ctx context.Context,
	runID int64,
	campID, customerID uuid.UUID,
	shardID int16,
	ledgerAmt, redisDelta int64,
	reason string,
) error {
	payload, err := json.Marshal(ReconciliationAdjustPayload{
		RunID:      runID,
		CampaignID: campID.String(),
		CustomerID: customerID.String(),
		ShardID:    shardID,
		LedgerAmt:  ledgerAmt,
		RedisDelta: redisDelta,
		Reason:     reason,
	})
	if err != nil {
		return err
	}
	q := db.New(w.svc.GetPool())
	_, err = q.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
		EventType: reconciliationAdjustEventType,
		Payload:   payload,
	})
	return err
}
