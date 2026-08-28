package reconciliation

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"time"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/metrics"
	"ad-event-processor/pkg/coldpath"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	redis "github.com/redis/go-redis/v9"
)

func parseReconciliationAdjustPayload(payload []byte) (ReconciliationAdjustPayload, error) {
	p, err := coldpath.UnmarshalStrict[ReconciliationAdjustPayload](payload)
	if err != nil {
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

func reconciliationAdjustIdempotencyHash(outboxEventID int64) string {
	return fmt.Sprintf("recon_adjust_outbox_%d", outboxEventID)
}

func reconciliationRedisAppliedKey(outboxEventID int64) string {
	return fmt.Sprintf("recon:redis_applied:%d", outboxEventID)
}

func postgresUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: true}
}

type ReconService struct {
	host Host
}

func NewReconService(host Host) *ReconService {
	return &ReconService{host: host}
}

func (s *ReconService) ReconcileWindow(ctx context.Context, start, end time.Time) error {
	opCtx, cancel := workerContext(ctx, workerBatchTimeout)
	defer cancel()

	run, err := s.createRun(opCtx, start, end)
	if err != nil {
		slog.Error("failed to create recon run record", "error", err, "start", start, "end", end)
		metrics.ReconRunsTotal.WithLabelValues("failed").Inc()
		return err
	}

	q := db.New(s.host.Pool())
	ledgerRows, err := q.SumLedgerSpendByCampaignWindowWithCustomer(ctx, db.SumLedgerSpendByCampaignWindowWithCustomerParams{
		CreatedAt:   pgtype.Timestamp{Time: start, Valid: true},
		CreatedAt_2: pgtype.Timestamp{Time: end, Valid: true},
	})
	if err != nil {
		s.failRun(opCtx, run.ID, err)
		metrics.ReconRunsTotal.WithLabelValues("failed").Inc()
		return err
	}

	type reconLedgerEntry struct {
		spent      int64
		customerID pgtype.UUID
	}
	ledgerMap := make(map[uuid.UUID]reconLedgerEntry, len(ledgerRows))
	for _, row := range ledgerRows {
		id, parseErr := uuid.FromBytes(row.CampaignID.Bytes[:])
		if parseErr != nil {
			slog.Error("failed to parse campaign id in recon run", "run_id", run.ID, "error", parseErr)
			continue
		}
		ledgerMap[id] = reconLedgerEntry{
			spent:      row.TotalSpentMicro,
			customerID: row.CustomerID,
		}
	}

	discrepancies := 0
	var totalDelta int64

	for campID, entry := range ledgerMap {
		ledgerSpent := entry.spent
		syncKey := domain.CampaignSyncKey(campID)
		redisClient := s.host.RedisClientForCampaign(campID)
		if redisClient == nil {
			slog.Error("no redis shard for campaign in recon", "campaign_id", campID)
			metrics.ReconAdjustmentErrors.Inc()
			continue
		}

		syncVal, err := redisClient.Get(opCtx, syncKey).Int64()
		if err != nil && !errors.Is(err, redis.Nil) {
			slog.Error("failed to fetch campaign sync budget from Redis in recon", "campaign_id", campID, "error", err)
			metrics.ReconAdjustmentErrors.Inc()
			s.failRun(opCtx, run.ID, err)
			metrics.ReconRunsTotal.WithLabelValues("failed").Inc()
			return err
		}

		delta := syncVal - ledgerSpent
		if delta == 0 {
			continue
		}

		customerID := entry.customerID

		discrepancies++

		_, err = s.host.Pool().Exec(opCtx, `
			INSERT INTO recon_discrepancies (run_id, campaign_id, customer_id, expected_spend, actual_spend, delta, redis_adjusted)
			VALUES ($1, $2, $3, $4, $5, $6, false)
		`, run.ID, domain.ToUUID(campID), customerID, syncVal, ledgerSpent, delta)
		if err != nil {
			slog.Error("failed to record recon discrepancy to postgres", "run_id", run.ID, "campaign_id", campID, "error", err)
			metrics.ReconAdjustmentErrors.Inc()
			s.failRun(opCtx, run.ID, err)
			metrics.ReconRunsTotal.WithLabelValues("failed").Inc()
			return err
		}

		chunkMicro := s.autoAdjustChunkMicro()
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

		shardID := int16(s.host.Sharder().GetShard(campID))
		custUUID, _ := uuid.FromBytes(customerID.Bytes[:])
		if err := s.enqueueReconciliationAdjust(opCtx, run.ID, campID, custUUID, shardID, -delta, delta, "hourly_window_recon"); err != nil {
			slog.Error("failed to enqueue recon adjustment", "campaign_id", campID, "delta", delta, "error", err)
			metrics.ReconAdjustmentErrors.Inc()
			continue
		}
		metrics.ReconCorrectionsTotal.Inc()
		totalDelta += delta
	}

	_, err = s.host.Pool().Exec(opCtx, `
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
	if discrepancies > 0 {
		if alerter := s.host.Alerter(); alerter != nil {
			alerter.AlertReconDiscrepancy(ctx,
				run.ID,
				discrepancies,
				totalDelta,
				start.Format(time.RFC3339)+"-"+end.Format(time.RFC3339),
			)
		}
	}
	return nil
}

func (s *ReconService) autoAdjustChunkMicro() int64 {
	cfg := s.host.Config()
	if cfg != nil && cfg.QuotaChunkSize > 0 {
		return cfg.QuotaChunkSize
	}
	return 5_000_000
}

func (s *ReconService) AlertStaleUnresolvedDiscrepancies(ctx context.Context) {
	alerter := s.host.Alerter()
	if alerter == nil {
		return
	}
	pool := s.host.Pool()
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
		alerter.AlertReconDiscrepancyUnresolved(ctx, runID, unresolved, totalDelta, period, oldest)
	}
}

func (s *ReconService) adjustRedisBudgetAtomically(ctx context.Context, redisClient redis.UniversalClient, campID uuid.UUID, delta int64) error {
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
	_, err := redisClient.Eval(ctx, script, []string{domain.CampaignSyncKey(campID)}, delta).Result()
	return err
}

func (s *ReconService) createRun(ctx context.Context, start, end time.Time) (struct{ ID int64 }, error) {
	var run struct{ ID int64 }
	err := s.host.Pool().QueryRow(ctx, `
		INSERT INTO recon_runs (period_start, period_end, status) VALUES ($1, $2, 'PENDING') RETURNING id
	`, start, end).Scan(&run.ID)
	return run, err
}

func (s *ReconService) enqueueReconciliationAdjust(
	ctx context.Context,
	runID int64,
	campID, customerID uuid.UUID,
	shardID int16,
	ledgerAmt, redisDelta int64,
	reason string,
) error {
	worker := &ReconWorker{host: s.host}
	return worker.enqueueReconciliationAdjust(ctx, runID, campID, customerID, shardID, ledgerAmt, redisDelta, reason)
}

func (s *ReconService) failRun(ctx context.Context, id int64, err error) {
	_, execErr := s.host.Pool().Exec(ctx, `UPDATE recon_runs SET status = 'FAILED' WHERE id = $1`, id)
	if execErr != nil {
		slog.Error("failed to mark recon run status as failed in postgres", "run_id", id, "error", execErr)
	}
	slog.Error("reconciliation run failed", "run_id", id, "error", err)
}

func (s *ReconService) reconciliationRedisAdjustApplied(
	ctx context.Context,
	redisClient redis.UniversalClient,
	eventID int64,
	p ReconciliationAdjustPayload,
	campID uuid.UUID,
) (bool, error) {
	if p.RunID > 0 {
		var adjusted bool
		err := s.host.Pool().QueryRow(ctx, `
			SELECT redis_adjusted FROM recon_discrepancies
			WHERE run_id = $1 AND campaign_id = $2`,
			p.RunID, domain.ToUUID(campID),
		).Scan(&adjusted)
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		return adjusted, nil
	}
	n, err := redisClient.Exists(ctx, reconciliationRedisAppliedKey(eventID)).Result()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func (s *ReconService) markReconciliationRedisAdjusted(
	ctx context.Context,
	redisClient redis.UniversalClient,
	eventID int64,
	p ReconciliationAdjustPayload,
	campID uuid.UUID,
) error {
	if p.RunID > 0 {
		_, err := s.host.Pool().Exec(ctx, `
			UPDATE recon_discrepancies
			SET redis_adjusted = true
			WHERE run_id = $1 AND campaign_id = $2`,
			p.RunID, domain.ToUUID(campID),
		)
		return err
	}
	return redisClient.Set(ctx, reconciliationRedisAppliedKey(eventID), "1", 7*24*time.Hour).Err()
}

func (s *ReconService) createSnapshotRun(ctx context.Context) (int64, error) {
	now := time.Now().UTC()
	var id int64
	err := s.host.Pool().QueryRow(ctx, `
		INSERT INTO recon_runs (period_start, period_end, status) VALUES ($1, $2, 'SNAPSHOT') RETURNING id`,
		now, now,
	).Scan(&id)
	return id, err
}

func abs(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}

const (
	hyg30ReconDriftThresholdMicro    = 1000
	hyg30ClickHouseStatsTolerancePct = 0.0001
	hyg30LedgerSampleSize            = 100
	reconciliationAdjustEventType    = "RECONCILIATION_ADJUST"
)

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
