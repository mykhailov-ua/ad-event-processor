package reconciliation

import (
	"context"
	"errors"
	"log/slog"
	"math"
	"time"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/governance"
	"ad-event-processor/internal/ledger"
	"ad-event-processor/internal/metrics"
	"ad-event-processor/internal/reports"
	"ad-event-processor/pkg/coldpath"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	redis "github.com/redis/go-redis/v9"
)

const workerBatchTimeout = 2 * time.Minute

func workerContext(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout <= 0 {
		return context.WithCancel(parent)
	}
	return context.WithTimeout(parent, timeout)
}

type ReconWorker struct {
	host     Host
	interval time.Duration
	quorum   *ShardQuorumTracker
}

func NewReconWorker(host Host, interval time.Duration) *ReconWorker {
	numShards := 1
	if host != nil {
		numShards = len(host.RedisShards())
	}
	return &ReconWorker{
		host:     host,
		interval: interval,
		quorum:   NewShardQuorumTracker(numShards, defaultDeadShardQuorum),
	}
}

func NewReconWorkerWithQuorum(host Host, interval, quorum time.Duration) *ReconWorker {
	w := NewReconWorker(host, interval)
	if w.quorum != nil && host != nil {
		w.quorum = NewShardQuorumTracker(len(host.RedisShards()), quorum)
	}
	return w
}

func (w *ReconWorker) Quorum() *ShardQuorumTracker {
	if w == nil {
		return nil
	}
	return w.quorum
}

func (w *ReconWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	quotaTicker := time.NewTicker(10 * time.Second)
	defer quotaTicker.Stop()

	drainCheckTicker := time.NewTicker(time.Minute)
	defer drainCheckTicker.Stop()

	snapshotTicker := time.NewTicker(reconSnapshotInterval(w.host.Config()))
	defer snapshotTicker.Stop()

	hyg30Interval := 5 * time.Minute
	if w.host.Config() != nil && w.host.Config().ReconHYG30IntervalMs > 0 {
		hyg30Interval = time.Duration(w.host.Config().ReconHYG30IntervalMs) * time.Millisecond
	}
	hyg30Ticker := time.NewTicker(hyg30Interval)
	defer hyg30Ticker.Stop()

	reconSvc := NewReconService(w.host)

	for {
		select {
		case <-ctx.Done():
			return
		case <-hyg30Ticker.C:
			w.runHYG30Audits(ctx)
		case <-snapshotTicker.C:
			if err := w.host.WithPostgresLow(ctx, func(runCtx context.Context) error {
				w.ReconcileBudgetSnapshot(runCtx)
				return nil
			}); err != nil && !errors.Is(err, ErrPostgresGateRejected) {
				slog.Error("budget snapshot recon failed", "err", err)
			}
		case <-ticker.C:
			// Hourly window lags 2h so ledger batch flush and stream settlement can land in PG.
			end := time.Now().Truncate(time.Hour).Add(-2 * time.Hour)
			start := end.Add(-time.Hour)
			if err := w.host.WithPostgresLow(ctx, func(runCtx context.Context) error {
				return reconSvc.ReconcileWindow(runCtx, start, end)
			}); err != nil && !errors.Is(err, ErrPostgresGateRejected) {
				slog.Error("recon worker iteration failed", "err", err, "window", start)
			}
		case <-quotaTicker.C:
			if w.host.Config() != nil && (w.host.Config().QuotaMode == "shadow" || w.host.Config().QuotaMode == "live") {
				if err := w.host.WithPostgresLow(ctx, func(runCtx context.Context) error {
					w.ReconcileQuotas(runCtx)
					return nil
				}); err != nil && !errors.Is(err, ErrPostgresGateRejected) {
					slog.Error("quota recon failed", "err", err)
				}
			}
		case <-drainCheckTicker.C:
			w.host.RunStuckDrainCheck(ctx)
			reconSvc.AlertStaleUnresolvedDiscrepancies(ctx)
		}
	}
}

func (w *ReconWorker) ReconcileQuotas(ctx context.Context) {
	if w == nil || w.host == nil {
		return
	}
	governance.NewQuotaRepairRunner(w.host, w.quorum).ReconcileQuotas(ctx)
}

func reconSnapshotInterval(cfg *config.Config) time.Duration {
	if cfg == nil {
		return 30 * time.Second
	}
	if cfg.Management.ReconSnapshotIntervalMs > 0 {
		return time.Duration(cfg.Management.ReconSnapshotIntervalMs) * time.Millisecond
	}
	return 30 * time.Second
}

func (w *ReconWorker) observeShardQuorum(ctx context.Context) {
	if w.quorum == nil {
		return
	}
	for shardIdx, redisClient := range w.host.RedisShards() {
		w.quorum.ObserveShard(ctx, shardIdx, redisClient)
	}
}

func (w *ReconWorker) runHYG30Audits(ctx context.Context) {
	if w == nil || w.host == nil {
		return
	}
	start := time.Now()
	pool := w.host.SettlementPool()
	if err := w.host.WithPostgresLow(ctx, func(runCtx context.Context) error {
		w.auditRedisPGLedger(runCtx, pool)
		w.auditPostgresClickHouseStats(runCtx)
		w.auditLedgerInvariantSample(runCtx, pool)
		return nil
	}); err != nil && !errors.Is(err, ErrPostgresGateRejected) {
		slog.Error("hyg30 recon audits failed", "error", err)
	}
	slog.Debug("hyg30 recon audits completed", "duration_ms", time.Since(start).Milliseconds())
}

// auditRedisPGLedger (HYG30-A): campaigns.current_spend vs balance_ledger sum vs Redis {id}:sync per shard.
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

	type hyg30AuditRow struct {
		campID, customerID         uuid.UUID
		postgresSpend, ledgerSpend int64
	}
	var auditRows []hyg30AuditRow
	for rows.Next() {
		var row hyg30AuditRow
		if err := rows.Scan(&row.campID, &row.customerID, &row.postgresSpend, &row.ledgerSpend); err != nil {
			continue
		}
		auditRows = append(auditRows, row)
	}
	if err := rows.Err(); err != nil {
		slog.Error("hyg30 audit A scan failed", "error", err)
		return
	}

	shards := w.host.RedisShards()
	byShard := make(map[int][]hyg30AuditRow)
	for _, row := range auditRows {
		shard := w.host.Sharder().GetShard(row.campID)
		if shard < 0 || shard >= len(shards) || shards[shard] == nil {
			continue
		}
		byShard[shard] = append(byShard[shard], row)
	}

	for shard, shardRows := range byShard {
		redisClient := shards[shard]
		pipe := redisClient.Pipeline()
		syncCmds := make(map[uuid.UUID]*redis.StringCmd, len(shardRows))
		for _, row := range shardRows {
			syncCmds[row.campID] = pipe.Get(ctx, domain.CampaignSyncKey(row.campID))
		}
		if _, err := pipe.Exec(ctx); err != nil && !errors.Is(err, redis.Nil) {
			slog.Error("hyg30 audit A redis pipeline failed", "shard", shard, "error", err)
			continue
		}
		for _, row := range shardRows {
			cmd := syncCmds[row.campID]
			redisSpend, err := cmd.Int64()
			if err != nil && !errors.Is(err, redis.Nil) {
				continue
			}
			w.processHYG30AuditDrift(ctx, row.campID, row.postgresSpend, row.ledgerSpend, redisSpend)
		}
	}
}

func (w *ReconWorker) processHYG30AuditDrift(ctx context.Context, campID uuid.UUID, postgresSpend, ledgerSpend, redisSpend int64) {
	drift := redisSpend - ledgerSpend
	if drift == 0 {
		metrics.ReconDriftMicro.DeleteLabelValues(campID.String())
		return
	}
	absDrift := drift
	if absDrift < 0 {
		absDrift = -absDrift
	}
	metrics.ReconDriftMicro.WithLabelValues(campID.String()).Set(float64(absDrift))
	if absDrift <= hyg30ReconDriftThresholdMicro {
		return
	}
	cfg := w.host.Config()
	if cfg != nil && cfg.ReconForceRefillEnabled() {
		if err := w.host.ForceRefillCampaignFromPG(ctx, campID, postgresSpend); err != nil {
			slog.Error("force refill from pg failed", "campaign_id", campID, "error", err)
		} else {
			slog.Info("force refill from pg applied", "campaign_id", campID, "postgres_spend", postgresSpend)
		}
	}
}

func (w *ReconWorker) auditPostgresClickHouseStats(ctx context.Context) {
	clickhouseQuery := w.host.ClickHouseQuery()
	if clickhouseQuery == nil {
		return
	}
	rows, err := w.host.Pool().Query(ctx, `
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

	type postgresStat struct {
		campID        uuid.UUID
		day           time.Time
		postgresTotal int64
	}
	var postgresStats []postgresStat
	var campaignIDs []uuid.UUID
	seenCamp := make(map[uuid.UUID]struct{})
	var minDay, maxDay time.Time

	for rows.Next() {
		var s postgresStat
		if err := rows.Scan(&s.campID, &s.day, &s.postgresTotal); err != nil {
			continue
		}
		if s.postgresTotal == 0 {
			continue
		}
		postgresStats = append(postgresStats, s)
		if _, ok := seenCamp[s.campID]; !ok {
			seenCamp[s.campID] = struct{}{}
			campaignIDs = append(campaignIDs, s.campID)
		}
		if minDay.IsZero() || s.day.Before(minDay) {
			minDay = s.day
		}
		if maxDay.IsZero() || s.day.After(maxDay) {
			maxDay = s.day
		}
	}
	if err := rows.Err(); err != nil {
		slog.Error("hyg30 audit B pg scan failed", "error", err)
		return
	}
	if len(postgresStats) == 0 || len(campaignIDs) == 0 {
		return
	}

	// HYG30-B: PG campaign_stats vs CH daily event totals; bounded by reports.ClickHouseQueryContext (10s).
	clickhouseCtx, cancel := clickhouseQueryContext(ctx)
	defer cancel()

	clickhouseTotals, err := reports.QueryClickHouseCampaignDailyEventTotals(clickhouseCtx, clickhouseQuery, campaignIDs, minDay, maxDay.Add(24*time.Hour))
	if err != nil {
		slog.Error("hyg30 audit B ch batch query failed", "error", err)
		return
	}

	for _, s := range postgresStats {
		key := reports.CampaignDailyTotalKey(s.campID, s.day)
		clickhouseTotal := clickhouseTotals[key]
		if clickhouseTotal == 0 {
			continue
		}
		diff := math.Abs(float64(int64(clickhouseTotal)-s.postgresTotal)) / float64(s.postgresTotal)
		if diff > hyg30ClickHouseStatsTolerancePct {
			slog.Warn("campaign stats stale vs clickhouse",
				"campaign_id", s.campID,
				"date", s.day.Format("2006-01-02"),
				"postgres_total", s.postgresTotal,
				"clickhouse_total", clickhouseTotal,
				"diff_pct", diff,
			)
		}
	}
}

// auditLedgerInvariantSample (HYG30-C): TABLESAMPLE customers; mismatch enqueues PAUSE_CAMPAIGN (AssertBudgetInvariant tier).
func (w *ReconWorker) auditLedgerInvariantSample(ctx context.Context, pool *pgxpool.Pool) {
	if pool == nil {
		return
	}
	rows, err := pool.Query(ctx, `
		SELECT id FROM customers TABLESAMPLE BERNOULLI (5) REPEATABLE (42)
		LIMIT $1`, hyg30LedgerSampleSize)
	if err != nil {
		slog.Error("hyg30 audit C sample failed", "error", err)
		return
	}
	defer rows.Close()

	var sampleIDs []uuid.UUID
	for rows.Next() {
		var customerID uuid.UUID
		if err := rows.Scan(&customerID); err != nil {
			continue
		}
		sampleIDs = append(sampleIDs, customerID)
	}
	if err := rows.Err(); err != nil {
		slog.Error("hyg30 audit C sample scan failed", "error", err)
		return
	}
	mismatches, err := ledger.ListLedgerInvariantMismatchesForIDs(ctx, pool, sampleIDs)
	if err != nil {
		slog.Error("hyg30 audit C invariant batch failed", "error", err)
		return
	}
	for _, customerID := range mismatches {
		slog.Error("ledger invariant failed for customer", "customer_id", customerID)
		w.enqueueForcePauseCustomer(ctx, customerID, ledger.ErrLedgerDrift.Error())
	}
}

func (w *ReconWorker) enqueueForcePauseCustomer(ctx context.Context, customerID uuid.UUID, reason string) {
	if w.host == nil || w.host.Pool() == nil {
		return
	}
	_ = reason
	camps, err := w.host.Pool().Query(ctx, `
		SELECT id FROM campaigns WHERE customer_id = $1 AND status = 'ACTIVE'`, customerID)
	if err != nil {
		return
	}
	defer camps.Close()

	var campIDs []uuid.UUID
	for camps.Next() {
		var campID uuid.UUID
		if err := camps.Scan(&campID); err != nil {
			continue
		}
		campIDs = append(campIDs, campID)
	}
	if err := camps.Err(); err != nil {
		return
	}
	if len(campIDs) == 0 {
		return
	}

	batch := &pgx.Batch{}
	for _, campID := range campIDs {
		payload, err := coldpath.MarshalOutbox(pauseCampaignPayload{CampaignID: campID.String()})
		if err != nil {
			continue
		}
		batch.Queue(`INSERT INTO outbox_events (event_type, payload) VALUES ($1, $2)`, "PAUSE_CAMPAIGN", payload)
	}
	br := w.host.Pool().SendBatch(ctx, batch)
	for range campIDs {
		if _, err := br.Exec(); err != nil {
			slog.Error("failed to enqueue FORCE_PAUSE batch", "error", err)
			_ = br.Close()
			return
		}
	}
	if err := br.Close(); err != nil {
		slog.Error("failed to close FORCE_PAUSE batch", "error", err)
	}
}

func (w *ReconWorker) ReconcileBudgetSnapshot(ctx context.Context) {
	if w == nil || w.host == nil || w.host.Pool() == nil {
		return
	}
	// Ping/replication/breaker quorum: skip dirty campaigns on shards confirmed dead.
	w.observeShardQuorum(ctx)

	reconSvc := NewReconService(w.host)
	campaignIDs, err := w.collectDirtyCampaignIDs(ctx)
	if err != nil {
		slog.Error("budget snapshot recon: dirty set scan failed", "error", err)
		return
	}

	postgresByID, err := w.loadCampaignBudgetPGBatch(ctx, campaignIDs)
	if err != nil {
		slog.Error("budget snapshot recon: batch pg load failed", "error", err)
		return
	}

	quotaMode := w.host.Config() != nil && (w.host.Config().QuotaMode == "shadow" || w.host.Config().QuotaMode == "live")
	snapByID := make(map[uuid.UUID]domain.BudgetReconSnapshot, len(campaignIDs))
	shards := w.host.RedisShards()
	byShard := make(map[int][]uuid.UUID)
	for _, campID := range campaignIDs {
		shardIdx := w.host.Sharder().GetShard(campID)
		if w.quorum != nil && w.quorum.DeadShardConfirmed(shardIdx) {
			continue
		}
		if shardIdx < 0 || shardIdx >= len(shards) || shards[shardIdx] == nil {
			continue
		}
		byShard[shardIdx] = append(byShard[shardIdx], campID)
	}
	for shardIdx, ids := range byShard {
		snaps, err := domain.BatchFetchBudgetReconSnapshots(ctx, shards[shardIdx], ids, quotaMode)
		if err != nil {
			slog.Error("budget snapshot recon: batch redis snapshot failed", "shard", shardIdx, "error", err)
			continue
		}
		for id, snap := range snaps {
			snapByID[id] = snap
		}
	}

	var checked, skipped, discrepancies int
	for _, campID := range campaignIDs {
		pg, postgresOk := postgresByID[campID]
		snap, snapOk := snapByID[campID]
		ok, disc, skip := w.reconcileCampaignSnapshot(ctx, reconSvc, campID, pg, snap, postgresOk, snapOk)
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
	for shardIdx, redisClient := range w.host.RedisShards() {
		if w.quorum != nil && w.quorum.DeadShardConfirmed(shardIdx) {
			continue
		}
		var cursor uint64
		for {
			keys, next, err := redisClient.SScan(ctx, "budget:dirty_campaigns", cursor, "", 200).Result()
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

func (w *ReconWorker) reconcileCampaignSnapshot(
	ctx context.Context,
	reconSvc *ReconService,
	campID uuid.UUID,
	pg campaignBudgetPG,
	snap domain.BudgetReconSnapshot,
	postgresOk, snapOk bool,
) (checked, discrepancy, skipped bool) {
	shardIdx := w.host.Sharder().GetShard(campID)
	if w.quorum != nil && w.quorum.DeadShardConfirmed(shardIdx) {
		return false, false, true
	}
	shards := w.host.RedisShards()
	if shardIdx >= len(shards) {
		return false, false, true
	}
	if !postgresOk {
		return false, false, true
	}
	if !snapOk {
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
	if deltas := w.host.BrokerDeltas(); deltas != nil {
		var brokerErr error
		// Broker mmap ring may hold spend not yet in Redis budget keys; include in redisTotal compare.
		brokerPending, brokerErr = deltas.PendingDeltaMicro(ctx, campID)
		if brokerErr != nil {
			slog.Warn("budget snapshot recon: broker pending delta unavailable",
				"campaign_id", campID, "error", brokerErr)
			return false, false, true
		}
	}

	postgresRemaining := pg.budgetLimit - pg.currentSpend
	redisTotal := snap.RedisBudgetRemainingTotal(brokerPending)
	drift := postgresRemaining - redisTotal
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

	_, err = w.host.Pool().Exec(ctx, `
		INSERT INTO recon_discrepancies (run_id, campaign_id, customer_id, expected_spend, actual_spend, delta, redis_adjusted)
		VALUES ($1, $2, $3, $4, $5, $6, false)`,
		runID, domain.ToUUID(campID), pgtype.UUID{Bytes: pg.customerID, Valid: true},
		redisTotal, postgresRemaining, drift,
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

func (w *ReconWorker) loadCampaignBudgetPGBatch(ctx context.Context, campIDs []uuid.UUID) (map[uuid.UUID]campaignBudgetPG, error) {
	out := make(map[uuid.UUID]campaignBudgetPG, len(campIDs))
	if len(campIDs) == 0 {
		return out, nil
	}
	pgIDs := make([]pgtype.UUID, len(campIDs))
	for i, id := range campIDs {
		pgIDs[i] = domain.ToUUID(id)
	}
	rows, err := w.host.Pool().Query(ctx, `
		SELECT c.id, c.customer_id, c.budget_limit, c.current_spend, c.updated_at,
		 COALESCE(q.reserved_amount, 0)
		FROM campaigns c
		LEFT JOIN campaign_quotas q ON q.campaign_id = c.id
		WHERE c.id = ANY($1::uuid[])`, pgIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var outID uuid.UUID
		var pg campaignBudgetPG
		if err := rows.Scan(&outID, &pg.customerID, &pg.budgetLimit, &pg.currentSpend, &pg.updatedAt, &pg.quotaReserved); err != nil {
			return nil, err
		}
		pg.campaignID = outID
		out[outID] = pg
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (w *ReconWorker) shouldSkipSnapshotGrace(snap domain.BudgetReconSnapshot, lastPGUpdate time.Time) bool {
	if snap.Inflight <= 0 {
		return false
	}
	grace := reconGraceWindow(w.host.Config())
	return time.Since(lastPGUpdate) < grace
}

func (w *ReconWorker) enqueueReconciliationAdjust(
	ctx context.Context,
	runID int64,
	campID, customerID uuid.UUID,
	shardID int16,
	ledgerAmt, redisDelta int64,
	reason string,
) error {
	payload, err := coldpath.MarshalOutbox(ReconciliationAdjustPayload{
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
	q := db.New(w.host.Pool())
	_, err = q.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
		EventType: reconciliationAdjustEventType,
		Payload:   payload,
	})
	return err
}
