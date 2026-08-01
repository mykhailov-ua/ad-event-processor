package controlplane

import (
	"bufio"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"espx/internal/config"
	"espx/internal/database"
	"espx/internal/dedup"
	"espx/internal/domain"
	db "espx/internal/domain/db"
	"espx/internal/ledger"
	billingdb "espx/internal/ledger/db"
	"espx/internal/licensing"
	"espx/internal/metrics"
	"espx/pkg/coldpath"
	"espx/pkg/dedupkey"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

const (
	workerBatchTimeout  = 2 * time.Minute
	workerDrainTimeout  = 30 * time.Second
	workerOutboxTimeout = 30 * time.Second
)

func workerContext(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if deadline, ok := parent.Deadline(); ok {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return context.WithCancel(parent)
		}
		if remaining < timeout {
			timeout = remaining
		}
	}
	return context.WithTimeout(parent, timeout)
}

type TLSImpersonationWorker struct {
	svc *Service
}

func NewTLSImpersonationWorker(svc *Service) *TLSImpersonationWorker {
	return &TLSImpersonationWorker{svc: svc}
}

func (w *TLSImpersonationWorker) Start(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	slog.Info("TLSImpersonationWorker started")

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.AnalyzeMismatches(ctx)
		}
	}
}

func (w *TLSImpersonationWorker) AnalyzeMismatches(ctx context.Context) {
	slog.Debug("TLSImpersonationWorker: analyzed TLS/UA mismatch metrics")
}

type AutoscaleBudgetWorker struct {
	svc         *Service
	syncWorkers []*domain.SyncWorker
}

func NewAutoscaleBudgetWorker(svc *Service, syncWorkers []*domain.SyncWorker) *AutoscaleBudgetWorker {
	return &AutoscaleBudgetWorker{
		svc:         svc,
		syncWorkers: syncWorkers,
	}
}

func (w *AutoscaleBudgetWorker) Start(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.svc.AutoscaleBudgets(ctx, w.syncWorkers); err != nil {
				slog.Error("autoscale budgets run failed", "err", err)
			}
		}
	}
}

type ScheduleWorker struct {
	svc      *Service
	interval time.Duration
}

func NewScheduleWorker(svc *Service) *ScheduleWorker {
	return &ScheduleWorker{svc: svc, interval: time.Minute}
}

func (w *ScheduleWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.svc.ProcessScheduleTick(ctx); err != nil {
				if database.IsShutdownError(err) {
					return
				}
				slog.Error("schedule worker tick failed", "err", err)
			}
		}
	}
}

type PacingControllerWorker struct {
	svc         *Service
	syncWorkers []*domain.SyncWorker
}

func NewPacingControllerWorker(svc *Service, syncWorkers []*domain.SyncWorker) *PacingControllerWorker {
	return &PacingControllerWorker{
		svc:         svc,
		syncWorkers: syncWorkers,
	}
}

func (w *PacingControllerWorker) Start(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.svc.ClosedLoopPacingController(ctx, w.syncWorkers); err != nil && !errors.Is(err, ErrPostgresGateRejected) {
				slog.Error("closed-loop pacing controller run failed", "err", err)
			}
			if err := w.svc.RunVPPPacingController(ctx); err != nil && !errors.Is(err, ErrPostgresGateRejected) {
				slog.Error("vpp pacing controller run failed", "err", err)
			}
		}
	}
}

type SystemStateWorker struct {
	svc *Service
}

func NewSystemStateWorker(svc *Service) *SystemStateWorker {
	return &SystemStateWorker{svc: svc}
}

func (w *SystemStateWorker) Start(ctx context.Context) {
	if w == nil || w.svc == nil {
		return
	}
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	if err := w.svc.SyncSystemState(ctx); err != nil {
		slog.Error("system state sync failed", "err", err)
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.svc.SyncSystemState(ctx); err != nil {
				slog.Error("system state sync failed", "err", err)
				continue
			}
			slog.Info("system state synchronized with redis")
		}
	}
}

type NodeCapacityScorerWorker struct {
	scorer *NodeCapacityScorer
}

func NewNodeCapacityScorerWorker(svc *Service) *NodeCapacityScorerWorker {
	return &NodeCapacityScorerWorker{scorer: NewNodeCapacityScorer(svc)}
}

func (w *NodeCapacityScorerWorker) Start(ctx context.Context) {
	if w == nil || w.scorer == nil {
		return
	}
	interval := 10 * time.Second
	if w.scorer.svc != nil && w.scorer.svc.cfg != nil && w.scorer.svc.cfg.UDPSyncIntervalMs > 0 {
		interval = time.Duration(w.scorer.svc.cfg.UDPSyncIntervalMs) * time.Millisecond
	}
	slog.Info("node capacity scorer starting", "region", w.scorer.region, "interval", interval)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	if err := w.scorer.Tick(ctx, time.Now().UTC()); err != nil {
		slog.Error("node capacity scorer tick failed", "err", err)
	}
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.scorer.Tick(ctx, time.Now().UTC()); err != nil {
				slog.Error("node capacity scorer tick failed", "err", err)
			}
		}
	}
}

type GlobalRegionTrafficScorerWorker struct {
	scorer *GlobalRegionTrafficScorer
}

func NewGlobalRegionTrafficScorerWorker(svc *Service) *GlobalRegionTrafficScorerWorker {
	return &GlobalRegionTrafficScorerWorker{scorer: NewGlobalRegionTrafficScorer(svc)}
}

func (w *GlobalRegionTrafficScorerWorker) Start(ctx context.Context) {
	if w == nil || w.scorer == nil {
		return
	}
	if w.scorer.svc == nil || w.scorer.svc.cfg == nil || !w.scorer.svc.cfg.MultiRegionGlobal() {
		return
	}
	interval := 10 * time.Second
	if w.scorer.svc.cfg.UDPSyncIntervalMs > 0 {
		interval = time.Duration(w.scorer.svc.cfg.UDPSyncIntervalMs) * time.Millisecond
	}
	slog.Info("global region traffic scorer starting", "interval", interval)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	if err := w.scorer.Tick(ctx, time.Now().UTC()); err != nil {
		slog.Error("global region traffic scorer tick failed", "err", err)
	}
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.scorer.Tick(ctx, time.Now().UTC()); err != nil {
				slog.Error("global region traffic scorer tick failed", "err", err)
			}
		}
	}
}

type DeliveryOptimizerWorker struct {
	svc         *Service
	syncWorkers []*domain.SyncWorker
	lastMABRun  time.Time
}

func NewDeliveryOptimizerWorker(svc *Service, syncWorkers []*domain.SyncWorker) *DeliveryOptimizerWorker {
	return &DeliveryOptimizerWorker{
		svc:         svc,
		syncWorkers: syncWorkers,
	}
}

func (w *DeliveryOptimizerWorker) Start(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.tick(ctx)
		}
	}
}

func (w *DeliveryOptimizerWorker) tick(ctx context.Context) {
	runMAB := false
	mabInterval := time.Duration(w.svc.cfg.MABIntervalMs) * time.Millisecond
	if mabInterval <= 0 {
		mabInterval = 15 * time.Minute
	}
	now := time.Now()
	if w.lastMABRun.IsZero() || now.Sub(w.lastMABRun) >= mabInterval {
		runMAB = true
		w.lastMABRun = now
	}

	if err := w.svc.RunDeliveryOptimizerTick(ctx, w.syncWorkers, runMAB); err != nil {
		slog.Error("delivery optimizer tick failed", "err", err, "run_mab", runMAB)
	}
}

type ErasureWorker struct {
	svc *Service
}

func NewErasureWorker(svc *Service) *ErasureWorker {
	return &ErasureWorker{svc: svc}
}

func (w *ErasureWorker) Start(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.svc.ProcessPrivacyErasureTick(ctx); err != nil {
				slog.Error("privacy erasure tick failed", "err", err)
			}
		}
	}
}

type ConsentRetentionWorker struct {
	svc *Service
}

func NewConsentRetentionWorker(svc *Service) *ConsentRetentionWorker {
	return &ConsentRetentionWorker{svc: svc}
}

func (w *ConsentRetentionWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.svc.CleanupConsentEvents(ctx); err != nil {
				slog.Error("consent retention cleanup failed", "err", err)
			}
		}
	}
}

type FloorOptimizerWorker struct {
	svc      *Service
	interval time.Duration
}

func NewFloorOptimizerWorker(svc *Service, interval time.Duration) *FloorOptimizerWorker {
	if interval <= 0 {
		interval = 7 * 24 * time.Hour
	}
	return &FloorOptimizerWorker{svc: svc, interval: interval}
}

func (s *Service) StartFloorOptimizerWorker(interval time.Duration) {
	if s == nil {
		return
	}
	w := NewFloorOptimizerWorker(s, interval)
	s.StartBackgroundWorker(func() {
		w.Start(s.ctx)
	})
}

func (w *FloorOptimizerWorker) Start(ctx context.Context) {
	if w == nil || w.svc == nil {
		return
	}
	slog.Info("floor optimizer worker starting", "interval", w.interval)
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	w.tick(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.tick(ctx)
		}
	}
}

func (w *FloorOptimizerWorker) tick(ctx context.Context) {
	n, err := w.svc.RunFloorOptimizer(ctx)
	if err != nil {
		slog.Error("floor optimizer tick failed", "err", err)
		return
	}
	slog.Info("floor optimizer tick complete", "placements", n)
}

const blacklistJanitorBatchSize = 200

type BlacklistJanitor struct {
	svc      *Service
	interval time.Duration
}

func NewBlacklistJanitor(svc *Service, interval time.Duration) *BlacklistJanitor {
	if interval <= 0 {
		interval = time.Minute
	}
	return &BlacklistJanitor{svc: svc, interval: interval}
}

func (j *BlacklistJanitor) Start(ctx context.Context) {
	if j == nil || j.svc == nil {
		return
	}

	ticker := time.NewTicker(j.interval)
	defer ticker.Stop()

	j.runOnce(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			j.runOnce(ctx)
		}
	}
}

func (j *BlacklistJanitor) runOnce(ctx context.Context) {
	opCtx, cancel := workerContext(ctx, workerBatchTimeout)
	defer cancel()

	rows, err := db.New(j.svc.GetPool()).ListExpiredBlacklistIPs(opCtx, blacklistJanitorBatchSize)
	if err != nil {
		slog.Error("blacklist janitor scan failed", "error", err)
		if j.svc.alerter != nil {
			j.svc.alerter.AlertBlacklistJanitorFailed(err)
		}
		return
	}
	if len(rows) == 0 {
		return
	}

	var removed int
	for _, row := range rows {
		if err := j.svc.UnblockIP(opCtx, row.Ip, row.Reason); err != nil {
			slog.Warn("blacklist janitor unblock failed",
				"ip", row.Ip,
				"reason", row.Reason,
				"error", err,
			)
			continue
		}
		removed++
	}

	slog.Info("blacklist janitor cycle complete",
		"expired_found", len(rows),
		"removed", removed,
	)
}

type UsageDailyFlushWorker struct {
	pool     *pgxpool.Pool
	interval time.Duration
}

func NewUsageDailyFlushWorker(pool *pgxpool.Pool, interval time.Duration) *UsageDailyFlushWorker {
	if interval <= 0 {
		interval = 24 * time.Hour
	}
	return &UsageDailyFlushWorker{pool: pool, interval: interval}
}

func (w *UsageDailyFlushWorker) Start(ctx context.Context) {
	if w == nil || w.pool == nil {
		return
	}
	slog.Info("usage daily flush worker starting", "interval", w.interval)
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.Flush(ctx, time.Now().UTC()); err != nil {
				slog.Error("usage daily flush failed", "err", err)
			}
		}
	}
}

func (w *UsageDailyFlushWorker) Flush(ctx context.Context, now time.Time) error {
	usageDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	period := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

	rows, err := w.pool.Query(ctx, `
		SELECT customer_id, meter, value
		FROM billing.usage_meters
		WHERE period = $1`, period)
	if err != nil {
		return err
	}
	defer rows.Close()

	var flushed int
	for rows.Next() {
		var custID uuid.UUID
		var meter string
		var value int64
		if err := rows.Scan(&custID, &meter, &value); err != nil {
			return err
		}
		_, err := w.pool.Exec(ctx, `
			INSERT INTO billing.usage_daily (customer_id, usage_date, meter, value)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (customer_id, usage_date, meter) DO UPDATE
			SET value = EXCLUDED.value`,
			custID, usageDate, meter, value)
		if err != nil {
			return err
		}
		flushed++
	}
	if flushed > 0 {
		slog.Info("usage daily flush complete", "date", usageDate.Format("2006-01-02"), "rows", flushed)
	}
	return rows.Err()
}

type LedgerInvariantWorker struct {
	pool     *pgxpool.Pool
	interval time.Duration
	notifier LedgerInvariantAlerter
}

type LedgerInvariantAlerter interface {
	AlertLedgerDrift(ctx context.Context, customerID string, err error)
}

func NewLedgerInvariantWorker(pool *pgxpool.Pool, cfg *config.Config, notifier LedgerInvariantAlerter) *LedgerInvariantWorker {
	hours := 24
	if cfg != nil && cfg.LedgerInvariantIntervalHours > 0 {
		hours = cfg.LedgerInvariantIntervalHours
	}
	return &LedgerInvariantWorker{
		pool:     pool,
		interval: time.Duration(hours) * time.Hour,
		notifier: notifier,
	}
}

func (w *LedgerInvariantWorker) Start(ctx context.Context) {
	if w == nil || w.pool == nil {
		return
	}
	slog.Info("ledger invariant worker starting", "interval", w.interval)
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.scanAll(ctx); err != nil {
				slog.Error("ledger invariant scan failed", "err", err)
			}
		}
	}
}

func (w *LedgerInvariantWorker) scanAll(ctx context.Context) error {
	rows, err := w.pool.Query(ctx, `SELECT id FROM customers`)
	if err != nil {
		return err
	}
	defer rows.Close()

	var mismatches int
	for rows.Next() {
		var customerID uuid.UUID
		if err := rows.Scan(&customerID); err != nil {
			continue
		}
		if err := ledger.CheckLedgerBalanceInvariant(ctx, w.pool, customerID); err != nil {
			mismatches++
			slog.Error("ledger invariant mismatch", "customer_id", customerID, "err", err)
			if w.notifier != nil {
				w.notifier.AlertLedgerDrift(ctx, customerID.String(), err)
			}
		}
	}
	if mismatches > 0 {
		return errors.New("ledger invariant mismatches detected")
	}
	return rows.Err()
}

const (
	eventsRetentionBatchSize = 10_000
	eventsRetentionBatchWait = 100 * time.Millisecond
)

type EventsRetentionWorker struct {
	queries db.Querier
	days    int
}

func NewEventsRetentionWorker(pool *pgxpool.Pool, retentionDays int) *EventsRetentionWorker {
	return &EventsRetentionWorker{
		queries: db.New(pool),
		days:    retentionDays,
	}
}

func (w *EventsRetentionWorker) Start(ctx context.Context) {
	if w == nil || w.days <= 0 {
		return
	}
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	w.runOnce(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.runOnce(ctx)
		}
	}
}

func (w *EventsRetentionWorker) RunOnce(ctx context.Context) int64 {
	return w.runOnce(ctx)
}

func (w *EventsRetentionWorker) runOnce(ctx context.Context) int64 {
	cutoff := time.Now().UTC().AddDate(0, 0, -w.days)
	var total int64

	for {
		deleted, err := w.queries.DeleteEventsOlderThanBatch(ctx, db.DeleteEventsOlderThanBatchParams{
			Cutoff:     pgtype.Timestamptz{Time: cutoff, Valid: true},
			BatchLimit: eventsRetentionBatchSize,
		})
		if err != nil {
			slog.Error("events retention batch failed", "err", err, "cutoff", cutoff)
			break
		}
		if deleted > 0 {
			metrics.EventsRetentionDeletedTotal.Add(float64(deleted))
			total += deleted
		}
		if deleted < eventsRetentionBatchSize {
			break
		}
		timer := time.NewTimer(eventsRetentionBatchWait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return total
		case <-timer.C:
		}
	}

	if total > 0 {
		slog.Info("events retention completed", "deleted", total, "retention_days", w.days, "cutoff", cutoff)
	}
	return total
}

type CampaignDrainWorker struct {
	svc *Service
}

func NewCampaignDrainWorker(svc *Service) *CampaignDrainWorker {
	return &CampaignDrainWorker{svc: svc}
}

func (w *CampaignDrainWorker) Start(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.svc.withPgHigh(ctx, func(runCtx context.Context) error {
				return w.ProcessDraining(runCtx)
			}); err != nil {
				if database.IsShutdownError(err) {
					return
				}
				slog.Error("failed to process draining campaigns", "err", err)
			}
		}
	}
}

func (w *CampaignDrainWorker) ProcessDraining(ctx context.Context) error {
	opCtx, cancel := workerContext(ctx, workerDrainTimeout)
	defer cancel()

	waitTimeoutMs := int64(100)
	if w.svc.cfg != nil && w.svc.cfg.Lifecycle.WaitTimeoutMs > 0 {
		waitTimeoutMs = int64(w.svc.cfg.Lifecycle.WaitTimeoutMs)
	}
	threshold := time.Now().Add(-time.Duration(waitTimeoutMs) * time.Millisecond)

	for i := 0; i < 100; i++ {
		finalized, err := w.finalizeNextDraining(opCtx, threshold)
		if err != nil {
			return err
		}
		if !finalized {
			return nil
		}
	}
	return nil
}

func (w *CampaignDrainWorker) finalizeNextDraining(ctx context.Context, threshold time.Time) (bool, error) {
	finalized := false
	err := pgx.BeginFunc(ctx, w.svc.GetPool(), func(tx pgx.Tx) error {
		q := db.New(tx)
		camps, err := q.GetDrainingCampaignsForUpdate(ctx, db.GetDrainingCampaignsForUpdateParams{
			UpdatedAt: pgtype.Timestamptz{Time: threshold, Valid: true},
			Limit:     1,
		})
		if err != nil {
			return err
		}
		if len(camps) == 0 {
			return nil
		}
		camp := camps[0]
		campaignID := uuid.UUID(camp.ID.Bytes)
		if err := w.svc.finalizeDrainingCampaign(ctx, q, campaignID, camp, "Finalized"); err != nil {
			return err
		}
		finalized = true
		return nil
	})
	return finalized, err
}

type CreditScoringWorker struct {
	svc *Service
}

func NewCreditScoringWorker(svc *Service) *CreditScoringWorker {
	return &CreditScoringWorker{svc: svc}
}

func (w *CreditScoringWorker) Start(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.EvaluateAll(ctx); err != nil {
				slog.Error("credit scoring evaluation failed", "err", err)
			}
		}
	}
}

func (w *CreditScoringWorker) EvaluateAll(ctx context.Context) error {
	opCtx, cancel := workerContext(ctx, workerBatchTimeout)
	defer cancel()

	queries := db.New(w.svc.GetPool())
	rows, err := queries.ListCustomersForScoring(opCtx)
	if err != nil {
		return err
	}

	for _, r := range rows {
		customerID := uuid.UUID(r.ID.Bytes)
		reconLag, err := queries.MaxCustomerReconLagMicro(opCtx, r.ID)
		if err != nil {
			slog.Error("failed to read recon lag for customer", "customer_id", customerID, "err", err)
			reconLag = 0
		}
		overdraft := w.calculateOverdraft(float64(r.AgeDays), r.TopupSum30d, reconLag)

		if err := w.svc.UpdateOverdraft(opCtx, customerID, overdraft); err != nil {
			slog.Error("failed to update overdraft for customer", "customer_id", customerID, "err", err)
		}
	}

	return nil
}

func (w *CreditScoringWorker) calculateOverdraft(ageDays float64, topupSum int64, reconLagMicro int64) int64 {
	if ageDays < w.svc.cfg.CreditScoringMinAgeDays {
		return 0
	}

	var overdraft int64
	if ageDays < w.svc.cfg.CreditScoringMatureAgeDays {
		overdraft = topupSum * w.svc.cfg.CreditScoringMidTierPercent / 100
	} else {
		overdraft = topupSum * w.svc.cfg.CreditScoringMaturePercent / 100
	}

	maxCap := w.svc.cfg.CreditScoringMaxCap
	if overdraft > maxCap {
		overdraft = maxCap
	}

	threshold := w.svc.cfg.CreditScoringReconLagThreshold
	if threshold > 0 && reconLagMicro > threshold {
		penalty := w.svc.cfg.CreditScoringReconLagPenaltyPct
		if penalty < 0 {
			penalty = 0
		}
		if penalty > 100 {
			penalty = 100
		}
		overdraft = overdraft * (100 - penalty) / 100
	}

	return overdraft
}

const supplyAuditInterval = 6 * time.Hour

type SupplyAuditWorker struct {
	svc      *Service
	interval time.Duration
}

func NewSupplyAuditWorker(svc *Service) *SupplyAuditWorker {
	return &SupplyAuditWorker{svc: svc, interval: supplyAuditInterval}
}

func (w *SupplyAuditWorker) Start(ctx context.Context) {
	if w == nil || w.svc == nil {
		return
	}
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	w.tick(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.tick(ctx)
		}
	}
}

func (w *SupplyAuditWorker) tick(ctx context.Context) {
	report, err := w.svc.AuditSupplyCompliance(ctx)
	if err != nil {
		if database.IsShutdownError(err) {
			return
		}
		slog.Error("supply audit failed", "err", err)
		return
	}
	if report.Issues > 0 {
		slog.Warn("supply audit found issues",
			"issues", report.Issues,
			"sellers", report.SellerCount,
			"ads_txt_lines", report.AdsTxtLines,
		)
	}
}

type SupplyAuditReport struct {
	SellerCount int `json:"seller_count"`
	AdsTxtLines int `json:"ads_txt_lines"`
	Issues      int `json:"issues"`
}

func (s *Service) AuditSupplyCompliance(ctx context.Context) (SupplyAuditReport, error) {
	out := SupplyAuditReport{}
	if s == nil || s.pool == nil {
		return out, nil
	}
	q := db.New(s.pool)
	sellers, err := q.ListSellers(ctx)
	if err != nil {
		return out, err
	}
	out.SellerCount = len(sellers)
	for _, row := range sellers {
		if row.Domain == "" || row.SellerID == "" {
			out.Issues++
		}
	}
	adsRows, err := q.ListAdsTxtEntries(ctx)
	if err != nil {
		return out, err
	}
	out.AdsTxtLines = len(adsRows)
	for _, row := range adsRows {
		if row.Domain == "" || row.PublisherAccountID == "" {
			out.Issues++
		}
	}
	if _, err := s.BuildSellersJSON(ctx); err != nil {
		out.Issues++
	}
	if _, err := s.BuildAdsTxt(ctx); err != nil {
		out.Issues++
	}
	return out, nil
}

type ReconWorker struct {
	svc      *Service
	interval time.Duration
	quorum   *ShardQuorumTracker
}

func NewReconWorker(svc *Service, interval time.Duration) *ReconWorker {
	numShards := 1
	if svc != nil {
		numShards = len(svc.rdbs)
	}
	return &ReconWorker{
		svc:      svc,
		interval: interval,
		quorum:   NewShardQuorumTracker(numShards, defaultDeadShardQuorum),
	}
}

func NewReconWorkerWithQuorum(svc *Service, interval, quorum time.Duration) *ReconWorker {
	w := NewReconWorker(svc, interval)
	if w.quorum != nil {
		w.quorum = NewShardQuorumTracker(len(svc.rdbs), quorum)
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

	snapshotTicker := time.NewTicker(reconSnapshotInterval(w.svc.cfg))
	defer snapshotTicker.Stop()

	hyg30Interval := 5 * time.Minute
	if w.svc.cfg != nil && w.svc.cfg.ReconHYG30IntervalMs > 0 {
		hyg30Interval = time.Duration(w.svc.cfg.ReconHYG30IntervalMs) * time.Millisecond
	}
	hyg30Ticker := time.NewTicker(hyg30Interval)
	defer hyg30Ticker.Stop()

	reconSvc := NewReconService(w.svc)

	for {
		select {
		case <-ctx.Done():
			return
		case <-hyg30Ticker.C:
			w.runHYG30Audits(ctx)
		case <-snapshotTicker.C:
			if err := w.svc.withPgLow(ctx, func(runCtx context.Context) error {
				w.ReconcileBudgetSnapshot(runCtx)
				return nil
			}); err != nil && !errors.Is(err, ErrPostgresGateRejected) {
				slog.Error("budget snapshot recon failed", "err", err)
			}
		case <-ticker.C:
			end := time.Now().Truncate(time.Hour).Add(-2 * time.Hour)
			start := end.Add(-time.Hour)
			if err := w.svc.withPgLow(ctx, func(runCtx context.Context) error {
				return reconSvc.ReconcileWindow(runCtx, start, end)
			}); err != nil && !errors.Is(err, ErrPostgresGateRejected) {
				slog.Error("recon worker iteration failed", "err", err, "window", start)
			}
		case <-quotaTicker.C:
			if w.svc.cfg != nil && (w.svc.cfg.QuotaMode == "shadow" || w.svc.cfg.QuotaMode == "live") {
				if err := w.svc.withPgLow(ctx, func(runCtx context.Context) error {
					w.ReconcileQuotas(runCtx)
					return nil
				}); err != nil && !errors.Is(err, ErrPostgresGateRejected) {
					slog.Error("quota recon failed", "err", err)
				}
			}
		case <-drainCheckTicker.C:
			w.svc.CheckStuckDrainJobs(ctx)
			reconSvc.AlertStaleUnresolvedDiscrepancies(ctx)
		}
	}
}

func (w *ReconWorker) ReconcileQuotas(ctx context.Context) {
	if w.svc == nil {
		return
	}
	w.observeShardQuorum(ctx)
	if w.svc.cfg != nil && w.svc.cfg.QuotaAutoRepair {
		w.RepairQuotaDrift(ctx)
	} else {
		w.MonitorQuotaDrift(ctx)
	}
}

func reconSnapshotInterval(cfg *config.Config) time.Duration {
	if cfg == nil {
		return 30 * time.Second
	}
	if cfg.Management.ReconSnapshotIntervalMs > 0 {
		return time.Duration(cfg.Management.ReconSnapshotIntervalMs) * time.Millisecond
	}
	ms := cfg.BudgetSyncIntervalMs
	if ms <= 0 {
		ms = 5000
	}
	return time.Duration(ms) * time.Millisecond
}

type NginxConfigWorker struct {
	svc        *Service
	exportPath string
}

func NewNginxConfigWorker(svc *Service, exportPath string) *NginxConfigWorker {
	return &NginxConfigWorker{
		svc:        svc,
		exportPath: exportPath,
	}
}

func (nginxWorker *NginxConfigWorker) Start(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := nginxWorker.ExportAndReload(ctx); err != nil {
				slog.Error("nginx export failed", "err", err)
			}
		}
	}
}

func (nginxWorker *NginxConfigWorker) ExportAndReload(ctx context.Context) error {
	if len(nginxWorker.svc.rdbs) == 0 {
		return fmt.Errorf("no redis client available")
	}

	var manual []string
	for _, rdb := range nginxWorker.svc.rdbs {
		m, err := rdb.SMembers(ctx, "blacklist:manual").Result()
		if err != nil {
			return fmt.Errorf("failed to fetch manual blacklist from shard: %w", err)
		}
		manual = append(manual, m...)
	}
	if err := nginxWorker.writeDenyFile("manual.conf", manual); err != nil {
		return err
	}

	var auto []string
	for _, rdb := range nginxWorker.svc.rdbs {
		a, err := rdb.SMembers(ctx, "blacklist:auto").Result()
		if err != nil {
			return fmt.Errorf("failed to fetch auto blacklist from shard: %w", err)
		}
		auto = append(auto, a...)
	}
	if err := nginxWorker.writeDenyFile("auto.conf", auto); err != nil {
		return err
	}

	flagPath := filepath.Join(nginxWorker.exportPath, "reload_required.flg")
	if err := os.WriteFile(flagPath, []byte("1\n"), 0644); err != nil {
		return fmt.Errorf("failed to write reload flag: %w", err)
	}

	slog.Info("nginx blacklist exported and reload signaled via flag file", "manual_count", len(manual), "auto_count", len(auto))
	return nil
}

func (nginxWorker *NginxConfigWorker) writeDenyFile(filename string, ips []string) (err error) {
	if err := os.MkdirAll(nginxWorker.exportPath, 0755); err != nil {
		return err
	}

	path := filepath.Join(nginxWorker.exportPath, filename)
	tmpPath := path + ".tmp"

	tmpFile, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to open temp config file: %w", err)
	}

	defer func() {
		if err != nil {
			_ = tmpFile.Close()
			_ = os.Remove(tmpPath)
		}
	}()

	bw := bufio.NewWriter(tmpFile)
	for _, ip := range ips {
		if ip == "" {
			continue
		}

		if net.ParseIP(ip) == nil {
			if _, _, errCIDR := net.ParseCIDR(ip); errCIDR != nil {
				slog.Warn("skipping invalid blacklist IP/CIDR to prevent injection", "ip", ip)
				continue
			}
		}

		if _, err = bw.WriteString("deny "); err != nil {
			return fmt.Errorf("failed to write directive prefix: %w", err)
		}
		if _, err = bw.WriteString(ip); err != nil {
			return fmt.Errorf("failed to write IP: %w", err)
		}
		if _, err = bw.WriteString(";\n"); err != nil {
			return fmt.Errorf("failed to write directive suffix: %w", err)
		}
	}

	if err = bw.Flush(); err != nil {
		return fmt.Errorf("failed to flush config buffer: %w", err)
	}

	if err = tmpFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync config file: %w", err)
	}

	if err = tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp config file: %w", err)
	}

	if err = os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("failed to atomically replace config file: %w", err)
	}

	return nil
}

const snapshotRunHourUTC = 0
const snapshotRunMinuteUTC = 15

type NodeMetricsSnapshotWorker struct {
	svc  *Service
	pool *pgxpool.Pool
}

func NewNodeMetricsSnapshotWorker(svc *Service) *NodeMetricsSnapshotWorker {
	return &NodeMetricsSnapshotWorker{
		svc:  svc,
		pool: svc.GetPool(),
	}
}

func (w *NodeMetricsSnapshotWorker) Start(ctx context.Context) {
	if w == nil || w.pool == nil {
		return
	}
	slog.Info("node metrics snapshot worker starting", "run_at_utc", "00:15")

	now := time.Now().UTC()
	if !now.Before(todaySnapshotRunAt(now)) {
		if err := w.RunOnce(ctx, snapshotDayFor(now)); err != nil {
			slog.Error("node metrics snapshot catch-up failed", "err", err)
		}
	}

	for {
		now := time.Now().UTC()
		wait := time.Until(nextSnapshotRunUTC(now))
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
			day := snapshotDayFor(time.Now().UTC())
			if err := w.RunOnce(ctx, day); err != nil {
				slog.Error("node metrics snapshot failed", "day", day.Format("2006-01-02"), "err", err)
			}
		}
	}
}

func (w *NodeMetricsSnapshotWorker) RunOnce(ctx context.Context, day time.Time) error {
	if w == nil || w.pool == nil {
		return nil
	}
	day = time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, time.UTC)
	run := func(runCtx context.Context) error {
		return w.snapshotDay(runCtx, day)
	}
	if w.svc != nil {
		return w.svc.withPgLow(ctx, run)
	}
	return run(ctx)
}

func (w *NodeMetricsSnapshotWorker) snapshotDay(ctx context.Context, day time.Time) error {
	start := day
	end := day.Add(24 * time.Hour)
	q := db.New(w.pool)

	rows, err := q.AggregateNodeMetricBucketsForDay(ctx, db.AggregateNodeMetricBucketsForDayParams{
		BucketTs:   pgtype.Timestamptz{Time: start, Valid: true},
		BucketTs_2: pgtype.Timestamptz{Time: end, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("aggregate node metric buckets day=%s: %w", day.Format("2006-01-02"), err)
	}

	dayParam := pgtype.Date{Time: day, Valid: true}
	for _, row := range rows {
		p99 := aggregateFloat(row.ValueP99)
		if err := q.UpsertNodeMetricDailySnapshot(ctx, db.UpsertNodeMetricDailySnapshotParams{
			Day:         dayParam,
			RegionCode:  row.RegionCode,
			Role:        row.Role,
			Metric:      row.Metric,
			ValueP50:    pgtype.Float8{Float64: row.ValueP50, Valid: true},
			ValueP99:    pgtype.Float8{Float64: p99, Valid: true},
			ValueMean:   pgtype.Float8{Float64: row.ValueMean, Valid: true},
			SampleCount: row.SampleCount,
		}); err != nil {
			return fmt.Errorf("upsert node metric daily snapshot day=%s metric=%s: %w",
				day.Format("2006-01-02"), row.Metric, err)
		}
	}

	slog.Info("node metric daily snapshots materialized",
		"day", day.Format("2006-01-02"),
		"rows", len(rows),
	)
	return nil
}

func snapshotDayFor(now time.Time) time.Time {
	d := now.UTC().AddDate(0, 0, -1)
	return time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.UTC)
}

func todaySnapshotRunAt(now time.Time) time.Time {
	n := now.UTC()
	return time.Date(n.Year(), n.Month(), n.Day(), snapshotRunHourUTC, snapshotRunMinuteUTC, 0, 0, time.UTC)
}

func nextSnapshotRunUTC(now time.Time) time.Time {
	runAt := todaySnapshotRunAt(now)
	if !now.Before(runAt) {
		runAt = runAt.Add(24 * time.Hour)
	}
	return runAt
}

func aggregateFloat(v interface{}) float64 {
	switch x := v.(type) {
	case float64:
		return x
	case float32:
		return float64(x)
	case int64:
		return float64(x)
	case int32:
		return float64(x)
	default:
		return 0
	}
}

const auditExportBatchSize = 1000

type AuditExportWorker struct {
	svc           *Service
	exportPath    string
	retentionDays int
}

func NewAuditExportWorker(svc *Service, exportPath string, retentionDays int) *AuditExportWorker {
	if retentionDays <= 0 {
		retentionDays = 90
	}
	return &AuditExportWorker{
		svc:           svc,
		exportPath:    exportPath,
		retentionDays: retentionDays,
	}
}

func (w *AuditExportWorker) Start(ctx context.Context, interval time.Duration) {
	if err := w.ExportDaily(ctx, time.Now().UTC()); err != nil {
		slog.Error("audit export failed", "err", err)
	}
	if err := w.cleanupOldExports(time.Now().UTC()); err != nil {
		slog.Error("audit export retention cleanup failed", "err", err)
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			now := time.Now().UTC()
			if err := w.ExportDaily(ctx, now); err != nil {
				slog.Error("audit export failed", "err", err)
			}
			if err := w.cleanupOldExports(now); err != nil {
				slog.Error("audit export retention cleanup failed", "err", err)
			}
		}
	}
}

func (w *AuditExportWorker) ExportDaily(ctx context.Context, now time.Time) error {
	day := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	return w.exportDay(ctx, day)
}

func (w *AuditExportWorker) exportDay(ctx context.Context, day time.Time) error {
	start := day
	end := day.Add(24 * time.Hour)
	filename := day.Format("2006-01-02") + ".csv"

	if err := os.MkdirAll(w.exportPath, 0755); err != nil {
		return fmt.Errorf("create audit export dir: %w", err)
	}

	path := filepath.Join(w.exportPath, filename)
	tmpPath := path + ".tmp"

	tmpFile, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("open audit export temp file: %w", err)
	}

	var rowsWritten int
	writeErr := func() error {
		cw := csv.NewWriter(tmpFile)
		if err := cw.Write([]string{"id", "admin_id", "action", "target_type", "target_id", "changes", "metadata", "created_at"}); err != nil {
			return fmt.Errorf("write audit export header: %w", err)
		}

		q := db.New(w.svc.GetPool())
		var offset int32
		for {
			batch, err := q.ListAuditLogsInRange(ctx, db.ListAuditLogsInRangeParams{
				CreatedAt:   pgtype.Timestamptz{Time: start, Valid: true},
				CreatedAt_2: pgtype.Timestamptz{Time: end, Valid: true},
				Limit:       auditExportBatchSize,
				Offset:      offset,
			})
			if err != nil {
				return fmt.Errorf("list audit logs for export: %w", err)
			}
			if len(batch) == 0 {
				break
			}

			for _, row := range batch {
				if err := cw.Write(auditLogCSVRecord(row)); err != nil {
					return fmt.Errorf("write audit export row: %w", err)
				}
				rowsWritten++
			}

			if len(batch) < auditExportBatchSize {
				break
			}
			offset += auditExportBatchSize
		}

		cw.Flush()
		if err := cw.Error(); err != nil {
			return fmt.Errorf("flush audit export csv: %w", err)
		}
		if err := tmpFile.Sync(); err != nil {
			return fmt.Errorf("sync audit export file: %w", err)
		}
		return nil
	}()

	if writeErr != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpPath)
		return writeErr
	}

	if err := tmpFile.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("close audit export temp file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("replace audit export file: %w", err)
	}

	slog.Info("audit log exported", "path", path, "day", filename, "rows", rowsWritten)
	return nil
}

func auditLogCSVRecord(row db.AdminAuditLog) []string {
	adminID := ""
	if row.AdminID.Valid {
		adminID = uuid.UUID(row.AdminID.Bytes).String()
	}
	targetID := ""
	if row.TargetID.Valid {
		targetID = uuid.UUID(row.TargetID.Bytes).String()
	}
	createdAt := ""
	if row.CreatedAt.Valid {
		createdAt = row.CreatedAt.Time.UTC().Format(time.RFC3339)
	}
	return []string{
		fmt.Sprintf("%d", row.ID),
		adminID,
		row.Action,
		row.TargetType,
		targetID,
		string(row.Changes),
		string(row.Metadata),
		createdAt,
	}
}

func (w *AuditExportWorker) cleanupOldExports(now time.Time) error {
	cutoff := now.AddDate(0, 0, -w.retentionDays)

	entries, err := os.ReadDir(w.exportPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read audit export dir: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".csv") {
			continue
		}
		dayStr := strings.TrimSuffix(entry.Name(), ".csv")
		day, err := time.ParseInLocation("2006-01-02", dayStr, time.UTC)
		if err != nil {
			continue
		}
		if day.Before(cutoff) {
			if err := os.Remove(filepath.Join(w.exportPath, entry.Name())); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("remove old audit export %s: %w", entry.Name(), err)
			}
			slog.Info("removed expired audit export", "file", entry.Name(), "retention_days", w.retentionDays)
		}
	}
	return nil
}

const (
	defaultNodeMetricsInterval = 10 * time.Second
	defaultNodeMetricsTTL      = 24 * time.Hour
)

type NodeMetricsWorker struct {
	svc      *Service
	pool     *pgxpool.Pool
	interval time.Duration
	ttl      time.Duration
	nodeID   string
	role     string
	region   int16
	acc      metricAccumulator
}

type metricAccumulator struct {
	mu      sync.Mutex
	samples map[string][]float64
}

func newMetricAccumulator() metricAccumulator {
	return metricAccumulator{samples: make(map[string][]float64)}
}

func (a *metricAccumulator) Record(metric string, value float64) {
	if metric == "" || math.IsNaN(value) || math.IsInf(value, 0) {
		return
	}
	a.mu.Lock()
	a.samples[metric] = append(a.samples[metric], value)
	a.mu.Unlock()
}

func (a *metricAccumulator) Drain() map[string][]float64 {
	a.mu.Lock()
	defer a.mu.Unlock()
	if len(a.samples) == 0 {
		return nil
	}
	out := a.samples
	a.samples = make(map[string][]float64)
	return out
}

func NewNodeMetricsWorker(svc *Service) *NodeMetricsWorker {
	nodeID, _ := os.Hostname()
	if svc != nil && svc.cfg != nil && svc.cfg.NodeID != "" {
		nodeID = svc.cfg.NodeID
	}
	role := "management"
	if svc != nil && svc.cfg != nil && svc.cfg.NodeRole != "" {
		role = svc.cfg.NodeRole
	}
	region := int16(0)
	if svc != nil && svc.cfg != nil {
		region = int16(svc.cfg.RegionCode)
	}
	return &NodeMetricsWorker{
		svc:      svc,
		pool:     svc.GetPool(),
		interval: defaultNodeMetricsInterval,
		ttl:      defaultNodeMetricsTTL,
		nodeID:   nodeID,
		role:     role,
		region:   region,
		acc:      newMetricAccumulator(),
	}
}

func (w *NodeMetricsWorker) Record(metric string, value float64) {
	if w == nil {
		return
	}
	w.acc.Record(metric, value)
}

func (w *NodeMetricsWorker) Start(ctx context.Context) {
	if w == nil || w.pool == nil {
		return
	}
	slog.Info("node metrics worker starting",
		"node_id", w.nodeID,
		"role", w.role,
		"region", w.region,
		"interval", w.interval,
	)
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.Flush(ctx, time.Now().UTC()); err != nil {
				slog.Error("node metrics flush failed", "node_id", w.nodeID, "err", err)
			}
		}
	}
}

func (w *NodeMetricsWorker) Flush(ctx context.Context, now time.Time) error {
	if w == nil || w.pool == nil {
		return nil
	}
	run := func(runCtx context.Context) error {
		if err := w.flushBuckets(runCtx, now); err != nil {
			return err
		}
		return w.expireBuckets(runCtx, now)
	}
	if w.svc != nil {
		return w.svc.withPgLow(ctx, run)
	}
	return run(ctx)
}

func (w *NodeMetricsWorker) flushBuckets(ctx context.Context, now time.Time) error {
	samples := w.acc.Drain()
	if len(samples) == 0 {
		return nil
	}
	bucketTS := now.Truncate(w.interval)
	q := db.New(w.pool)
	for metric, values := range samples {
		p50, p99, mean, count := aggregateSamples(values)
		if count == 0 {
			continue
		}
		if err := q.InsertNodeMetricBucket(ctx, db.InsertNodeMetricBucketParams{
			NodeID:      w.nodeID,
			RegionCode:  w.region,
			Role:        w.role,
			BucketTs:    pgtype.Timestamptz{Time: bucketTS, Valid: true},
			Metric:      metric,
			ValueP50:    pgtype.Float8{Float64: p50, Valid: true},
			ValueP99:    pgtype.Float8{Float64: p99, Valid: true},
			ValueMean:   pgtype.Float8{Float64: mean, Valid: true},
			SampleCount: count,
		}); err != nil {
			return fmt.Errorf("flush node metric bucket node=%s: %w", w.nodeID, err)
		}
	}
	return nil
}

func (w *NodeMetricsWorker) expireBuckets(ctx context.Context, now time.Time) error {
	cutoff := now.Add(-w.ttl)
	q := db.New(w.pool)
	if _, err := q.DeleteExpiredNodeMetricBuckets(ctx, pgtype.Timestamptz{Time: cutoff, Valid: true}); err != nil {
		return fmt.Errorf("expire node metric buckets node=%s: %w", w.nodeID, err)
	}
	return nil
}

func aggregateSamples(values []float64) (p50, p99, mean float64, count int64) {
	if len(values) == 0 {
		return 0, 0, 0, 0
	}
	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)
	count = int64(len(sorted))
	var sum float64
	for _, v := range sorted {
		sum += v
	}
	mean = sum / float64(count)
	p50 = percentile(sorted, 0.50)
	p99 = percentile(sorted, 0.99)
	return p50, p99, mean, count
}

func percentile(sorted []float64, q float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	if q <= 0 {
		return sorted[0]
	}
	if q >= 1 {
		return sorted[len(sorted)-1]
	}
	pos := q * float64(len(sorted)-1)
	lo := int(math.Floor(pos))
	hi := int(math.Ceil(pos))
	if lo == hi {
		return sorted[lo]
	}
	frac := pos - float64(lo)
	return sorted[lo]*(1-frac) + sorted[hi]*frac
}

const (
	meterBillableEvents = "events"
	meterAcceptedEvents = "accepted_events"
	volumeMeterSourcePG = "pg"
	volumeMeterSourceCH = "ch"
)

type VolumeMeterWorker struct {
	pool     *pgxpool.Pool
	ch       *database.CHQuery
	source   string
	interval time.Duration
	pgGate   *PostgresGate
}

func NewVolumeMeterWorker(pool *pgxpool.Pool, ch *database.CHQuery, source string, interval time.Duration, pgGate *PostgresGate) *VolumeMeterWorker {
	if interval <= 0 {
		interval = time.Hour
	}
	if source == "" {
		source = volumeMeterSourcePG
	}
	return &VolumeMeterWorker{pool: pool, ch: ch, source: source, interval: interval, pgGate: pgGate}
}

func (w *VolumeMeterWorker) Start(ctx context.Context) {
	if w == nil || w.pool == nil {
		return
	}
	if w.source == volumeMeterSourceCH && w.ch == nil {
		slog.Warn("volume meter source=ch but clickhouse query is nil, worker not started")
		return
	}
	slog.Info("volume meter worker starting", "interval", w.interval, "source", w.source)
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.RunHour(ctx, time.Now().UTC()); err != nil {
				slog.Error("volume meter rollup failed", "err", err)
			}
		}
	}
}

type rollupRow struct {
	CampaignID uuid.UUID
	EventType  string
	Count      uint64
}

type pgMeterRow struct {
	CustomerID uuid.UUID
	Count      int64
}

func (w *VolumeMeterWorker) RunHour(ctx context.Context, now time.Time) error {
	if w.pgGate != nil {
		if err := w.pgGate.AcquireLow(ctx); err != nil {
			return err
		}
		defer w.pgGate.ReleaseLow()
	}
	return w.runHour(ctx, now)
}

func (w *VolumeMeterWorker) runHour(ctx context.Context, now time.Time) error {
	hourEnd := now.Truncate(time.Hour)
	hourStart := hourEnd.Add(-time.Hour)
	period := time.Date(hourStart.Year(), hourStart.Month(), 1, 0, 0, 0, 0, time.UTC)

	if w.source == volumeMeterSourcePG {
		return w.runPGHour(ctx, hourStart, hourEnd, period)
	}
	return w.runCHHour(ctx, hourStart, hourEnd, period)
}

func (w *VolumeMeterWorker) runPGHour(ctx context.Context, hourStart, hourEnd, period time.Time) error {
	rows, err := w.queryPGRollups(ctx, hourStart, hourEnd)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return nil
	}

	q := billingdb.New(w.pool)
	var customers int
	for _, row := range rows {
		if row.Count <= 0 {
			continue
		}
		if _, err := q.IncrementUsageMeter(ctx, billingdb.IncrementUsageMeterParams{
			CustomerID: pgtype.UUID{Bytes: row.CustomerID, Valid: true},
			Meter:      meterAcceptedEvents,
			Period:     pgtype.Date{Time: period, Valid: true},
			Value:      row.Count,
		}); err != nil {
			return fmt.Errorf("increment usage meter customer=%s: %w", row.CustomerID, err)
		}
		customers++
	}
	metrics.VolumeMeterRowsTotal.Add(float64(len(rows)))
	slog.Info("volume meter pg rollup complete",
		"hour", hourStart.Format(time.RFC3339),
		"customers", customers,
		"rows", len(rows),
	)
	return nil
}

func (w *VolumeMeterWorker) queryPGRollups(ctx context.Context, from, to time.Time) ([]pgMeterRow, error) {
	const q = `
		SELECT c.customer_id, COUNT(*)::bigint AS cnt
		FROM events e
		JOIN campaigns c ON c.id = e.campaign_id
		WHERE e.created_at >= $1 AND e.created_at < $2 AND e.status = 'accepted'
		GROUP BY c.customer_id`

	pgRows, err := w.pool.Query(ctx, q, from, to)
	if err != nil {
		return nil, fmt.Errorf("pg volume meter query: %w", err)
	}
	defer pgRows.Close()

	var out []pgMeterRow
	for pgRows.Next() {
		var row pgMeterRow
		if err := pgRows.Scan(&row.CustomerID, &row.Count); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, pgRows.Err()
}

func (w *VolumeMeterWorker) runCHHour(ctx context.Context, hourStart, hourEnd, period time.Time) error {
	rows, err := w.queryCHRollups(ctx, hourStart, hourEnd)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return nil
	}

	campaignCustomers, err := w.loadCampaignCustomers(ctx)
	if err != nil {
		return err
	}

	customerUnits := ComputeWeightedUnitsFromRows(rows, campaignCustomers)
	q := billingdb.New(w.pool)
	for custID, units := range customerUnits {
		if units <= 0 {
			continue
		}
		if _, err := q.IncrementUsageMeter(ctx, billingdb.IncrementUsageMeterParams{
			CustomerID: pgtype.UUID{Bytes: custID, Valid: true},
			Meter:      meterBillableEvents,
			Period:     pgtype.Date{Time: period, Valid: true},
			Value:      units,
		}); err != nil {
			return fmt.Errorf("increment usage meter customer=%s: %w", custID, err)
		}
	}
	metrics.VolumeMeterRowsTotal.Add(float64(len(customerUnits)))
	slog.Info("volume meter ch rollup complete",
		"hour", hourStart.Format(time.RFC3339),
		"customers", len(customerUnits),
	)
	return nil
}

func (w *VolumeMeterWorker) queryCHRollups(ctx context.Context, from, to time.Time) ([]rollupRow, error) {
	const q = `
		SELECT
			campaign_id,
			event_type,
			sum(event_count) AS cnt
		FROM ad_event_processor.audit_log_rollups
		WHERE rollup_hour >= ? AND rollup_hour < ?
		GROUP BY campaign_id, event_type`

	chRows, err := w.ch.Query(ctx, q, from, to)
	if err != nil {
		return nil, fmt.Errorf("clickhouse rollup query: %w", err)
	}
	defer chRows.Close()

	var out []rollupRow
	for chRows.Next() {
		var row rollupRow
		if err := chRows.Scan(&row.CampaignID, &row.EventType, &row.Count); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, chRows.Err()
}

func (w *VolumeMeterWorker) loadCampaignCustomers(ctx context.Context) (map[uuid.UUID]uuid.UUID, error) {
	pgRows, err := w.pool.Query(ctx, `SELECT id, customer_id FROM campaigns`)
	if err != nil {
		return nil, err
	}
	defer pgRows.Close()

	out := make(map[uuid.UUID]uuid.UUID)
	for pgRows.Next() {
		var campID, custID uuid.UUID
		if err := pgRows.Scan(&campID, &custID); err != nil {
			return nil, err
		}
		out[campID] = custID
	}
	return out, pgRows.Err()
}

func ComputeWeightedUnitsFromRows(rows []rollupRow, campaignCustomers map[uuid.UUID]uuid.UUID) map[uuid.UUID]int64 {
	customerUnits := make(map[uuid.UUID]int64)
	for _, row := range rows {
		custID, ok := campaignCustomers[row.CampaignID]
		if !ok {
			continue
		}
		cat := licensing.ClassifyEventType(row.EventType)
		units := int64(row.Count) * licensing.BillableWeightPermille(cat) / 1000
		customerUnits[custID] += units
	}
	return customerUnits
}

type FraudModelVersionPayload struct {
	ModelVersion string `json:"model_version"`
	Hash         string `json:"hash"`
	ShardID      int    `json:"shard_id"`
}

type FraudModelSyncOrchestrator struct {
	svc *Service
}

func NewFraudModelSyncOrchestrator(svc *Service) *FraudModelSyncOrchestrator {
	return &FraudModelSyncOrchestrator{svc: svc}
}

type FraudModelSyncWorker struct {
	orchestrator *FraudModelSyncOrchestrator
	interval     time.Duration
}

func NewFraudModelSyncWorker(svc *Service, interval time.Duration) *FraudModelSyncWorker {
	if interval <= 0 {
		interval = 30 * time.Second
	}
	return &FraudModelSyncWorker{
		orchestrator: NewFraudModelSyncOrchestrator(svc),
		interval:     interval,
	}
}

func (w *FraudModelSyncWorker) Start(ctx context.Context) {
	if w == nil || w.orchestrator == nil {
		return
	}
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.orchestrator.Tick(ctx); err != nil {
				slog.Error("fraud model sync tick failed", "err", err)
			}
			if err := w.orchestrator.svc.CheckAndHandleStaleEpochs(ctx); err != nil {
				slog.Warn("fraud model stale epoch handling", "err", err)
			}
		}
	}
}

func (o *FraudModelSyncOrchestrator) Tick(ctx context.Context) error {
	pool := o.svc.GetPool()
	if pool == nil {
		return fmt.Errorf("postgres pool not available")
	}

	var versionID, artifactHash string
	err := pool.QueryRow(ctx, "SELECT id, artifact_hash FROM ml_model_versions WHERE status = 'SYNCING' LIMIT 1").Scan(&versionID, &artifactHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("failed to query syncing model version: %w", err)
	}

	numShards := len(o.svc.rdbs)
	if numShards == 0 {
		return fmt.Errorf("no redis shards configured")
	}

	rows, err := pool.Query(ctx, "SELECT shard_id, phase, started_at FROM ml_shard_sync_state WHERE model_version = $1", versionID)
	if err != nil {
		return fmt.Errorf("failed to query shard sync states: %w", err)
	}
	defer rows.Close()

	type shardState struct {
		phase     string
		startedAt time.Time
	}
	states := make(map[int]shardState)
	for rows.Next() {
		var shardID int
		var phase string
		var startedAt time.Time
		if err := rows.Scan(&shardID, &phase, &startedAt); err != nil {
			return fmt.Errorf("failed to scan shard sync state: %w", err)
		}
		states[shardID] = shardState{phase: phase, startedAt: startedAt}
	}

	var activeSyncShard = -1
	for id, state := range states {
		if state.phase == "SYNC" {
			activeSyncShard = id
			break
		}
	}

	if activeSyncShard != -1 {
		state := states[activeSyncShard]
		if time.Since(state.startedAt) > 180*time.Second {
			slog.Error("fraud model sync timed out on shard, triggering rollback", "shard_id", activeSyncShard, "version", versionID)
			return o.rollbackShard(ctx, activeSyncShard, versionID)
		}

		passed, err := o.runCanaryCheck(ctx, activeSyncShard, versionID)
		if err != nil {
			slog.Warn("fraud model sync canary check failed with error, rolling back", "shard_id", activeSyncShard, "version", versionID, "err", err)
			return o.rollbackShard(ctx, activeSyncShard, versionID)
		}

		if passed {
			slog.Info("fraud model sync canary passed, cutting over shard to ACTIVE", "shard_id", activeSyncShard, "version", versionID)
			_, err = pool.Exec(ctx, "UPDATE ml_shard_sync_state SET phase = 'ACTIVE' WHERE shard_id = $1 AND model_version = $2", activeSyncShard, versionID)
			if err != nil {
				return fmt.Errorf("failed to update shard phase to ACTIVE: %w", err)
			}

			rdb := o.svc.rdbs[activeSyncShard]
			if rdb != nil {
				rdb.Set(ctx, "ml:model:version", versionID, 0)
				rdb.Set(ctx, "ml:model:hash", artifactHash, 0)
				rdb.Set(ctx, "ml:model:applied_at", time.Now().Unix(), 0)
			}
		} else {
			slog.Warn("fraud model sync canary failed (high FP rate), rolling back", "shard_id", activeSyncShard, "version", versionID)
			return o.rollbackShard(ctx, activeSyncShard, versionID)
		}

		return nil
	}

	var nextShardToSync = -1
	for i := 0; i < numShards; i++ {
		state, exists := states[i]
		if !exists || state.phase == "ROLLBACK" {
			nextShardToSync = i
			break
		}
	}

	if nextShardToSync != -1 {
		slog.Info("fraud model sync starting on shard", "shard_id", nextShardToSync, "version", versionID)
		_, err = pool.Exec(ctx, `
			INSERT INTO ml_shard_sync_state (shard_id, model_version, phase, started_at)
			VALUES ($1, $2, 'SYNC', NOW())
			ON CONFLICT (shard_id, model_version) DO UPDATE SET phase = 'SYNC', started_at = NOW()`,
			nextShardToSync, versionID)
		if err != nil {
			return fmt.Errorf("failed to insert shard sync state: %w", err)
		}

		payload, err := coldpath.MarshalJSON(FraudModelVersionPayload{
			ModelVersion: versionID,
			Hash:         artifactHash,
			ShardID:      nextShardToSync,
		})
		if err != nil {
			return err
		}

		_, err = db.New(pool).CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
			EventType: "ML_MODEL_VERSION",
			Payload:   payload,
		})
		return err
	}

	slog.Info("fraud model sync complete on all shards, activating version globally", "version", versionID)
	return pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, "UPDATE ml_model_versions SET status = 'ACTIVE' WHERE id = $1", versionID)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, "UPDATE ml_model_versions SET status = 'RETIRED' WHERE id <> $1 AND status = 'ACTIVE'", versionID)
		if err != nil {
			return err
		}
		return nil
	})
}

func (o *FraudModelSyncOrchestrator) rollbackShard(ctx context.Context, shardID int, versionID string) error {
	pool := o.svc.GetPool()
	_, err := pool.Exec(ctx, "UPDATE ml_shard_sync_state SET phase = 'ROLLBACK' WHERE shard_id = $1 AND model_version = $2", shardID, versionID)
	if err != nil {
		return fmt.Errorf("failed to update shard phase to ROLLBACK: %w", err)
	}

	var prevVersionID, prevHash string
	err = pool.QueryRow(ctx, "SELECT id, artifact_hash FROM ml_model_versions WHERE status = 'ACTIVE' LIMIT 1").Scan(&prevVersionID, &prevHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			rdb := o.svc.rdbs[shardID]
			if rdb != nil {
				rdb.Del(ctx, "ml:model:version", "ml:model:hash", "ml:model:applied_at")
			}
			return nil
		}
		return fmt.Errorf("failed to query previous active model version: %w", err)
	}

	rdb := o.svc.rdbs[shardID]
	if rdb != nil {
		rdb.Set(ctx, "ml:model:version", prevVersionID, 0)
		rdb.Set(ctx, "ml:model:hash", prevHash, 0)
		rdb.Set(ctx, "ml:model:applied_at", time.Now().Unix(), 0)
	}

	return nil
}

func (o *FraudModelSyncOrchestrator) runCanaryCheck(ctx context.Context, shardID int, versionID string) (bool, error) {
	if o.svc.chQuery == nil {
		return true, nil
	}

	query := `
		SELECT window_start, ip_address, campaign_id, events, clicks, spend_micro, budget_limit_micro, unique_users, unique_uas
		FROM ad_event_processor.ml_features_1m
		WHERE window_start >= now() - INTERVAL 1 HOUR
		LIMIT 1000`

	rows, err := o.svc.chQuery.Query(ctx, query)
	if err != nil {
		return false, fmt.Errorf("clickhouse query failed: %w", err)
	}
	defer rows.Close()

	var totalRows int
	var highScores int

	for rows.Next() {
		var windowStart time.Time
		var ipAddress, campaignID string
		var events, clicks, uniqueUsers, uniqueUAs uint64
		var spendMicro, budgetLimitMicro int64
		if err := rows.Scan(&windowStart, &ipAddress, &campaignID, &events, &clicks, &spendMicro, &budgetLimitMicro, &uniqueUsers, &uniqueUAs); err != nil {
			return false, fmt.Errorf("clickhouse scan failed: %w", err)
		}
		totalRows++

		if clicks > 10 && clicks*2 > events {
			highScores++
		}
	}

	if totalRows == 0 {
		return true, nil
	}

	fpRate := float64(highScores) / float64(totalRows)
	slog.Info("fraud model sync canary stats", "shard_id", shardID, "total_rows", totalRows, "high_scores", highScores, "fp_rate", fpRate)

	if fpRate > 0.10 {
		return false, nil
	}

	return true, nil
}

type OutboxWorker struct {
	svc *Service
}

func NewOutboxWorker(svc *Service) *OutboxWorker {
	return &OutboxWorker{svc: svc}
}

type CampaignPayload struct {
	CampaignID  string `json:"campaign_id"`
	BudgetLimit int64  `json:"budget_limit,omitempty"`
}

type SettingsPayload struct {
	Settings map[string]string `json:"settings"`
}

type BlacklistPayload struct {
	Action string `json:"action"`
	IP     string `json:"ip"`
	Reason string `json:"reason"`
}

type FraudThreatPayload struct {
	Action     string  `json:"action"`
	IP         string  `json:"ip"`
	CampaignID string  `json:"campaign_id"`
	Score      float64 `json:"score"`
	Boost      int32   `json:"boost"`
	TTLSeconds int64   `json:"ttl_seconds"`
}

func normalizeBlacklistReason(reason string) string {
	if reason == "" {
		return "manual"
	}
	return reason
}

func (worker *OutboxWorker) Start(ctx context.Context, interval time.Duration) {
	if err := worker.ProcessOutbox(ctx); err != nil {
		slog.Error("outbox startup cold sync failed", "err", err)
	}

	slog.Info("outbox worker starting polling loop", "interval", interval)

	pollBackoff := newOutboxPollBackoff()
	pollTimer := time.NewTimer(interval)
	defer pollTimer.Stop()

	recoveryTicker := time.NewTicker(interval * 5)
	defer recoveryTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-recoveryTicker.C:
			worker.reclaimStaleProcessing(ctx)
			worker.recordOutboxLagMetrics(ctx)
		case <-pollTimer.C:
			var processed int
			var err error
			if worker.svc != nil {
				err = worker.svc.withPgHigh(ctx, func(runCtx context.Context) error {
					var innerErr error
					processed, innerErr = worker.ProcessOutboxWithCount(runCtx, 1000)
					return innerErr
				})
			} else {
				processed, err = worker.ProcessOutboxWithCount(ctx, 1000)
			}
			worker.recordOutboxLagMetrics(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				if database.IsShutdownError(err) {
					return
				}
				slog.Error("outbox polling loop iteration failed, retrying in 2s", "err", err)
				pollTimer.Reset(2 * time.Second)
				continue
			}

			pollTimer.Reset(pollBackoff.next(processed))
		}
	}
}

func (worker *OutboxWorker) reclaimStaleProcessing(ctx context.Context) {
	_, err := worker.svc.GetPool().Exec(ctx, `
		UPDATE outbox_events
		SET status = 'PENDING', processing_started_at = NULL
		WHERE status = 'PROCESSING'
		  AND processing_started_at IS NOT NULL
		  AND processing_started_at < NOW() - INTERVAL '1 minute'`)
	if err != nil && ctx.Err() == nil && !database.IsShutdownError(err) {
		slog.Error("failed to reclaim stale outbox events", "err", err)
	}
}

func (worker *OutboxWorker) ProcessOutbox(ctx context.Context) error {
	_, err := worker.ProcessOutboxWithCount(ctx, 1000)
	return err
}

func (worker *OutboxWorker) ProcessOutboxWithCount(ctx context.Context, limit int32) (int, error) {
	opCtx, cancel := workerContext(ctx, workerOutboxTimeout)
	defer cancel()

	var events []db.OutboxEvent

	err := pgx.BeginFunc(opCtx, worker.svc.GetPool(), func(tx pgx.Tx) error {
		q := db.New(tx)
		var err error
		events, err = q.GetPendingOutboxEventsForUpdate(opCtx, limit)
		if err != nil || len(events) == 0 {
			return err
		}

		ids := make([]int64, len(events))
		for i, ev := range events {
			ids[i] = ev.ID
		}

		_, err = tx.Exec(opCtx, `
			UPDATE outbox_events
			SET status = 'PROCESSING', processing_started_at = NOW()
			WHERE id = ANY($1)`, ids)
		if err != nil {
			return err
		}
		return nil
	})

	if err != nil || len(events) == 0 {
		return 0, err
	}

	processedIDs := make([]int64, 0, len(events))
	revertIDs := make([]int64, 0, len(events))
	var batchErrs []error

	for _, ev := range events {
		if err := worker.handleOutboxEvent(opCtx, ctx, ev); err != nil {
			slog.Warn("redis outbox processing failed for event, marking for revert", "id", ev.ID, "err", err)
			revertIDs = append(revertIDs, ev.ID)
			batchErrs = append(batchErrs, fmt.Errorf("outbox event %d: %w", ev.ID, err))
			continue
		}
		processedIDs = append(processedIDs, ev.ID)
	}

	if len(processedIDs) > 0 {
		_, err = worker.svc.GetPool().Exec(opCtx, "UPDATE outbox_events SET status = 'PROCESSED' WHERE id = ANY($1)", processedIDs)
		if err != nil {
			slog.Error("failed to mark outbox events as processed", "err", err)
			batchErrs = append(batchErrs, fmt.Errorf("mark outbox processed: %w", err))
		}
	}

	if len(revertIDs) > 0 {
		_, err = worker.svc.GetPool().Exec(opCtx, `
			UPDATE outbox_events
			SET status = 'PENDING', processing_started_at = NULL
			WHERE id = ANY($1)`, revertIDs)
		if err != nil {
			slog.Error("failed to revert failed outbox events", "err", err)
			batchErrs = append(batchErrs, fmt.Errorf("revert outbox failed: %w", err))
		}
	}

	if len(batchErrs) > 0 {
		return len(processedIDs), errors.Join(batchErrs...)
	}

	return len(processedIDs), nil
}

func (worker *OutboxWorker) campaignRemainingBudget(ctx context.Context, campaignID uuid.UUID) (int64, error) {
	var limit, spend int64
	err := worker.svc.GetPool().QueryRow(ctx, `
		SELECT budget_limit, current_spend
		FROM campaigns
		WHERE id = $1`, domain.ToUUID(campaignID)).Scan(&limit, &spend)
	if err != nil {
		return 0, err
	}
	remaining := limit - spend
	if remaining < 0 {
		remaining = 0
	}
	return remaining, nil
}

func (worker *OutboxWorker) setCampaignBudgetRemaining(ctx context.Context, pipe redis.Pipeliner, campaignIDStr string, campaignID uuid.UUID, payloadLimit int64) error {
	remaining, err := worker.campaignRemainingBudget(ctx, campaignID)
	if err != nil {
		if payloadLimit <= 0 {
			return err
		}
		remaining = payloadLimit
	}
	if remaining <= 0 {
		return nil
	}
	pipe.Set(ctx, fmt.Sprintf("budget:campaign:%s", campaignIDStr), remaining, 0)
	return nil
}

func ToUUID(u uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: u, Valid: true}
}

const fraudQuarantineChannel = "fraud:quarantine"
const blacklistUpdateChannel = "blacklist:update"

func (worker *OutboxWorker) applyBlacklistPayload(ctx context.Context, p BlacklistPayload, queuedAt time.Time) error {
	if len(worker.svc.rdbs) == 0 {
		return fmt.Errorf("no redis client available")
	}
	reason := normalizeBlacklistReason(p.Reason)
	key := "blacklist:" + reason
	add := p.Action == "add"
	if p.Action != "add" && p.Action != "remove" {
		return fmt.Errorf("unknown blacklist action: %s", p.Action)
	}
	if err := syncGlobalSetMemberToAllShards(ctx, worker.svc.rdbs, key, p.IP, add); err != nil {
		return fmt.Errorf("blacklist sync failed: %w", err)
	}
	if reason == "fraud" && p.Action == "add" {
		_ = publishControlChannelToAllShards(ctx, worker.svc.rdbs, fraudQuarantineChannel, p.IP)
	}
	_ = publishControlChannelToAllShards(ctx, worker.svc.rdbs, blacklistUpdateChannel, p.IP+":"+reason)
	if !queuedAt.IsZero() {
		lag := time.Since(queuedAt).Seconds()
		if lag >= 0 {
			metrics.BlacklistReplicationLag.Observe(lag)
		}
	}
	return nil
}

func (worker *OutboxWorker) syncBrandCreativesToRedis(ctx context.Context, brandIDStr string) error {
	brandID, err := coldpath.ParseUUID(brandIDStr)
	if err != nil {
		return err
	}
	rows, err := db.New(worker.svc.GetPool()).ListActiveBrandCreatives(ctx, ToUUID(brandID))
	if err != nil {
		return err
	}
	type creativeEntry struct {
		ID     string `json:"id"`
		URL    string `json:"url"`
		Weight int32  `json:"weight"`
	}
	entries := make([]creativeEntry, len(rows))
	for i, r := range rows {
		entries[i] = creativeEntry{
			ID:     uuid.UUID(r.ID.Bytes).String(),
			URL:    r.LandingUrl,
			Weight: r.Weight,
		}
	}
	payload, err := coldpath.MarshalJSON(entries)
	if err != nil {
		return err
	}
	if len(worker.svc.rdbs) == 0 {
		return fmt.Errorf("no redis client")
	}
	key := "brand:creatives:" + brandIDStr
	for _, rdb := range worker.svc.rdbs {
		if err := rdb.Set(ctx, key, payload, 0).Err(); err != nil {
			return err
		}
	}
	return nil
}

const (
	defaultOpLeaseJanitorPeriod = 5 * time.Second
	defaultOpLeasePollInterval  = 200 * time.Millisecond
	defaultOpLeaseExpireBatch   = int32(500)
)

type OperationLeaseBookRequest struct {
	OpID         uuid.UUID
	RegionCode   int16
	Role         string
	ReplicaSetID uuid.UUID
	Attempt      int32
	FactorU      uuid.UUID
	Scope        dedupkey.Scope
	ReplicaNodes []string
	BookAckNodes []string
}

type OperationLeaseBookResult struct {
	Lease     db.OperationLease
	AckCount  int32
	Quorum    int32
	QuorumMet bool
}

type OperationLeaseExecuteFunc func(ctx context.Context, lease db.OperationLease, claim dedup.ClaimResult) error

type OperationLeaseWorker struct {
	svc            *Service
	nodeID         string
	role           string
	region         int16
	timeoutSec     int
	maxRenewals    int32
	janitorPeriod  time.Duration
	pollInterval   time.Duration
	fencing        *LeaseFencingRegistry
	opKeyGate      OpKeyPoolGate
	executor       OperationLeaseExecuteFunc
	renewHeartbeat bool
	onRenew        LeaseRenewHook
}

func NewOperationLeaseWorker(svc *Service) *OperationLeaseWorker {
	nodeID, _ := os.Hostname()
	timeoutSec := 30
	maxRenewals := int32(3)
	role := "management"
	region := int16(0)
	fencingDir := ""
	if svc != nil && svc.cfg != nil {
		if svc.cfg.NodeID != "" {
			nodeID = svc.cfg.NodeID
		}
		if svc.cfg.NodeRole != "" {
			role = svc.cfg.NodeRole
		}
		region = int16(svc.cfg.RegionCode)
		if svc.cfg.OpLeaseTimeoutSec > 0 {
			timeoutSec = svc.cfg.OpLeaseTimeoutSec
		}
		if svc.cfg.OpLeaseMaxRenewals > 0 {
			maxRenewals = int32(svc.cfg.OpLeaseMaxRenewals)
		}
		fencingDir = svc.cfg.OpLeaseFencingDir
	}
	if fencingDir == "" {
		fencingDir = filepath.Join(os.TempDir(), "espx-op-lease", nodeID)
	}
	fencing, err := NewLeaseFencingRegistry(fencingDir)
	if err != nil {
		slog.Warn("operation lease fencing registry init failed", "dir", fencingDir, "err", err)
	}
	return &OperationLeaseWorker{
		svc:            svc,
		nodeID:         nodeID,
		role:           role,
		region:         region,
		timeoutSec:     timeoutSec,
		maxRenewals:    maxRenewals,
		janitorPeriod:  defaultOpLeaseJanitorPeriod,
		pollInterval:   defaultOpLeasePollInterval,
		fencing:        fencing,
		renewHeartbeat: true,
	}
}

func (w *OperationLeaseWorker) SetExecutor(fn OperationLeaseExecuteFunc) {
	if w == nil {
		return
	}
	w.executor = fn
}

func (w *OperationLeaseWorker) SetOpKeyPoolGate(gate OpKeyPoolGate) {
	if w == nil {
		return
	}
	w.opKeyGate = gate
}

func (w *OperationLeaseWorker) Book(ctx context.Context, req OperationLeaseBookRequest) (OperationLeaseBookResult, error) {
	empty := OperationLeaseBookResult{}
	if w == nil || w.svc == nil || w.svc.pool == nil {
		return empty, fmt.Errorf("operation lease book op_id=%s: worker unavailable", req.OpID)
	}
	if w.opKeyGate != nil && w.opKeyGate.ShouldShed() {
		return empty, ErrOpKeyPoolShed
	}
	if req.OpID == uuid.Nil {
		return empty, fmt.Errorf("operation lease book: op_id required")
	}
	if req.ReplicaSetID == uuid.Nil {
		return empty, fmt.Errorf("operation lease book op_id=%s: replica_set_id required", req.OpID)
	}
	if req.Attempt <= 0 {
		req.Attempt = 1
	}
	if len(req.ReplicaNodes) == 0 {
		req.ReplicaNodes = []string{w.nodeID}
	}
	ackNodes := req.BookAckNodes
	if len(ackNodes) == 0 && len(req.ReplicaNodes) == 1 {
		ackNodes = req.ReplicaNodes
	}

	scopeRaw, err := EncodeLeaseDedupScope(req.Scope, req.Attempt)
	if err != nil {
		return empty, fmt.Errorf("operation lease book op_id=%s: %w", req.OpID, err)
	}

	if !w.pgAvailable(ctx) {
		return w.bookRedis(ctx, req)
	}

	var lease db.OperationLease
	err = w.svc.withPgHigh(ctx, func(runCtx context.Context) error {
		q := db.New(w.svc.pool)
		booked, err := q.BookOperationLease(runCtx, db.BookOperationLeaseParams{
			OpID:         domain.ToUUID(req.OpID),
			RegionCode:   req.RegionCode,
			Role:         req.Role,
			ReplicaSetID: domain.ToUUID(req.ReplicaSetID),
			Attempt:      req.Attempt,
			FactorU:      domain.ToUUID(req.FactorU),
			DedupScope:   scopeRaw,
			TimeoutSec:   int32(w.timeoutSec),
		})
		if err != nil {
			return err
		}
		for _, nodeID := range req.ReplicaNodes {
			if err := q.InsertOperationLeaseReplica(runCtx, db.InsertOperationLeaseReplicaParams{
				OpID:   domain.ToUUID(req.OpID),
				NodeID: nodeID,
			}); err != nil {
				return err
			}
		}
		for _, nodeID := range ackNodes {
			if err := q.UpsertOperationLeaseReplicaBookAck(runCtx, db.UpsertOperationLeaseReplicaBookAckParams{
				OpID:   domain.ToUUID(req.OpID),
				NodeID: nodeID,
			}); err != nil {
				return err
			}
		}
		lease = booked
		return nil
	})
	if err != nil {
		if !w.pgAvailable(ctx) || isPgUnavailable(err) {
			return w.bookRedis(ctx, req)
		}
		return empty, fmt.Errorf("operation lease book op_id=%s: %w", req.OpID, err)
	}

	result, qerr := w.quorumStatus(ctx, req.OpID)
	if qerr != nil {
		return empty, fmt.Errorf("operation lease book op_id=%s: %w", req.OpID, qerr)
	}
	result.Lease = lease
	if result.QuorumMet {
		metrics.OpLeaseBookedTotal.Inc()
		return result, nil
	}
	return result, ErrLeaseQuorumNotMet
}

func (w *OperationLeaseWorker) AckBook(ctx context.Context, opID uuid.UUID, nodeID string) (OperationLeaseBookResult, error) {
	empty := OperationLeaseBookResult{}
	if w == nil || w.svc == nil || w.svc.pool == nil {
		return empty, fmt.Errorf("operation lease ack book op_id=%s: worker unavailable", opID)
	}
	if nodeID == "" {
		nodeID = w.nodeID
	}
	if !w.pgAvailable(ctx) {
		return w.ackBookRedis(ctx, opID, nodeID, 3)
	}
	err := w.svc.withPgHigh(ctx, func(runCtx context.Context) error {
		return db.New(w.svc.pool).UpsertOperationLeaseReplicaBookAck(runCtx, db.UpsertOperationLeaseReplicaBookAckParams{
			OpID:   domain.ToUUID(opID),
			NodeID: nodeID,
		})
	})
	if err != nil {
		if !w.pgAvailable(ctx) || isPgUnavailable(err) {
			return w.ackBookRedis(ctx, opID, nodeID, 3)
		}
		return empty, fmt.Errorf("operation lease ack book op_id=%s: %w", opID, err)
	}
	result, err := w.quorumStatus(ctx, opID)
	if err != nil {
		return empty, err
	}
	if result.QuorumMet {
		metrics.OpLeaseBookedTotal.Inc()
	}
	return result, nil
}

func (w *OperationLeaseWorker) quorumStatus(ctx context.Context, opID uuid.UUID) (OperationLeaseBookResult, error) {
	if !w.pgAvailable(ctx) {
		return w.quorumStatusRedis(ctx, opID, 3)
	}
	q := db.New(w.svc.pool)
	opUUID := domain.ToUUID(opID)
	replicaCount, err := q.CountOperationLeaseReplicas(ctx, opUUID)
	if err != nil {
		if isPgUnavailable(err) {
			return w.quorumStatusRedis(ctx, opID, 3)
		}
		return OperationLeaseBookResult{}, err
	}
	ackCount, err := q.CountOperationLeaseBookAcks(ctx, opUUID)
	if err != nil {
		if isPgUnavailable(err) {
			return w.quorumStatusRedis(ctx, opID, int(replicaCount))
		}
		return OperationLeaseBookResult{}, err
	}
	quorum := QuorumRequired(int(replicaCount))
	return OperationLeaseBookResult{
		AckCount:  ackCount,
		Quorum:    quorum,
		QuorumMet: ackCount >= quorum,
	}, nil
}

func (w *OperationLeaseWorker) ExecuteOp(ctx context.Context, opID uuid.UUID, execute OperationLeaseExecuteFunc) error {
	if w == nil || w.svc == nil || w.svc.pool == nil {
		return fmt.Errorf("operation lease execute op_id=%s: worker unavailable", opID)
	}
	if execute == nil {
		return fmt.Errorf("operation lease execute op_id=%s: executor required", opID)
	}

	quorum, err := w.quorumStatus(ctx, opID)
	if err != nil {
		return fmt.Errorf("operation lease execute op_id=%s: %w", opID, err)
	}
	if !quorum.QuorumMet {
		return nil
	}

	opUUID := domain.ToUUID(opID)
	preLease, err := db.New(w.svc.pool).GetOperationLease(ctx, opUUID)
	if err != nil {
		return fmt.Errorf("operation lease execute op_id=%s: %w", opID, err)
	}
	replicaSetID := uuid.UUID(preLease.ReplicaSetID.Bytes)

	fencingEpoch, err := w.nextFencingEpoch(replicaSetID)
	if err != nil {
		return fmt.Errorf("operation lease execute op_id=%s: %w", opID, err)
	}

	var won bool
	err = w.svc.withPgHigh(ctx, func(runCtx context.Context) error {
		q := db.New(w.svc.pool)
		rows, err := q.OperationLeaseClaimExecuting(runCtx, db.OperationLeaseClaimExecutingParams{
			OpID:         opUUID,
			NodeID:       w.nodeID,
			FencingEpoch: fencingEpoch,
		})
		if err != nil {
			return err
		}
		won = len(rows) > 0
		return nil
	})
	if err != nil {
		return fmt.Errorf("operation lease execute op_id=%s: %w", opID, err)
	}
	if !won {
		return nil
	}
	metrics.OpLeaseExecutionTotal.Inc()

	lease, err := db.New(w.svc.pool).GetOperationLease(ctx, opUUID)
	if err != nil {
		return fmt.Errorf("operation lease execute op_id=%s: %w", opID, err)
	}
	if err := AuthoritativeLeaseView(lease, w.nodeID, time.Now().UTC()); err != nil {
		return fmt.Errorf("operation lease execute op_id=%s: %w", opID, err)
	}
	if err := w.fencing.Validate(replicaSetID, lease.FencingEpoch); err != nil {
		return fmt.Errorf("operation lease execute op_id=%s: %w", opID, err)
	}

	scope, err := DedupScopeForLease(lease)
	if err != nil {
		return fmt.Errorf("operation lease execute op_id=%s: %w", opID, err)
	}
	factorU := uuid.UUID(lease.FactorU.Bytes)

	adapter := w.svc.dedupAdapter()
	if adapter == nil {
		return fmt.Errorf("operation lease execute op_id=%s: dedup adapter unavailable", opID)
	}

	claim, err := adapter.ClaimConfirm(ctx, scope, factorU)
	if err != nil {
		return fmt.Errorf("operation lease execute op_id=%s: %w", opID, err)
	}
	if guardErr := dedup.GuardOutcome(claim); guardErr != nil {
		return fmt.Errorf("operation lease execute op_id=%s: %w", opID, guardErr)
	}

	applySideEffects := claim.ShouldApply()
	if claim.Outcome == dedup.OutcomeAlreadyConfirmed {
		resume, resumeErr := adapter.NeedsResumeApply(ctx, claim.DedupKey)
		if resumeErr != nil {
			return fmt.Errorf("operation lease execute op_id=%s: %w", opID, resumeErr)
		}
		applySideEffects = resume
	}
	if applySideEffects {
		lease, err = db.New(w.svc.pool).GetOperationLease(ctx, opUUID)
		if err != nil {
			return fmt.Errorf("operation lease execute op_id=%s: %w", opID, err)
		}
		if err := AuthoritativeLeaseView(lease, w.nodeID, time.Now().UTC()); err != nil {
			return fmt.Errorf("operation lease execute op_id=%s: %w", opID, err)
		}
		if err := w.fencing.Validate(replicaSetID, lease.FencingEpoch); err != nil {
			return fmt.Errorf("operation lease execute op_id=%s: %w", opID, err)
		}
		done := make(chan struct{})
		if w.renewHeartbeat {
			go w.runRenewHeartbeat(ctx, opID, done)
		}
		err = execute(ctx, lease, claim)
		close(done)
		if err != nil {
			return fmt.Errorf("operation lease execute op_id=%s: %w", opID, err)
		}
		if err := adapter.RecordApply(ctx, claim.DedupKey); err != nil {
			return fmt.Errorf("operation lease execute op_id=%s: %w", opID, err)
		}
	}

	_, err = db.New(w.svc.pool).CompleteOperationLease(ctx, opUUID)
	if err != nil {
		return fmt.Errorf("operation lease execute op_id=%s: %w", opID, err)
	}
	return nil
}

func (w *OperationLeaseWorker) RenewLease(ctx context.Context, opID uuid.UUID) (db.OperationLease, error) {
	if w == nil || w.svc == nil || w.svc.pool == nil {
		return db.OperationLease{}, fmt.Errorf("operation lease renew op_id=%s: worker unavailable", opID)
	}
	var renewed db.OperationLease
	err := w.svc.withPgHigh(ctx, func(runCtx context.Context) error {
		row, err := db.New(w.svc.pool).RenewOperationLease(runCtx, db.RenewOperationLeaseParams{
			OpID:           domain.ToUUID(opID),
			TimeoutSec:     int32(w.timeoutSec),
			ExecutorNodeID: pgtype.Text{String: w.nodeID, Valid: true},
			MaxRenewals:    w.maxRenewals,
		})
		if err != nil {
			return err
		}
		renewed = row
		return nil
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.OperationLease{}, ErrLeaseRenewExhausted
		}
		return db.OperationLease{}, fmt.Errorf("operation lease renew op_id=%s: %w", opID, err)
	}
	metrics.OpLeaseHeartbeatRenewTotal.Inc()
	if w.onRenew != nil {
		w.onRenew(opID)
	}
	return renewed, nil
}

func (w *OperationLeaseWorker) nextFencingEpoch(replicaSetID uuid.UUID) (int64, error) {
	if w.fencing == nil {
		return 1, nil
	}
	return w.fencing.Next(replicaSetID)
}

func (w *OperationLeaseWorker) RunJanitor(ctx context.Context) (int32, error) {
	if w == nil || w.svc == nil || w.svc.pool == nil {
		return 0, nil
	}
	ok, err := w.tryJanitorLock(ctx)
	if err != nil {
		return 0, fmt.Errorf("operation lease janitor: %w", err)
	}
	if !ok {
		return 0, nil
	}
	defer w.releaseJanitorLock(ctx)

	var expired int32
	err = w.svc.withPgLow(ctx, func(runCtx context.Context) error {
		n, err := db.New(w.svc.pool).OperationLeaseExpireStale(runCtx, defaultOpLeaseExpireBatch)
		if err != nil {
			return err
		}
		expired = n
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("operation lease janitor: %w", err)
	}
	if expired > 0 {
		metrics.OpLeaseExpiredTotal.Add(float64(expired))
	}
	return expired, nil
}

func (w *OperationLeaseWorker) tryJanitorLock(ctx context.Context) (bool, error) {
	var ok bool
	err := w.svc.pool.QueryRow(ctx,
		`SELECT pg_try_advisory_lock($1::int, $2::int)`,
		opLeaseJanitorLockKey1, int32(w.region),
	).Scan(&ok)
	return ok, err
}

func (w *OperationLeaseWorker) releaseJanitorLock(ctx context.Context) {
	_, _ = w.svc.pool.Exec(ctx,
		`SELECT pg_advisory_unlock($1::int, $2::int)`,
		opLeaseJanitorLockKey1, int32(w.region),
	)
}

func (w *OperationLeaseWorker) ProcessBooked(ctx context.Context) error {
	if w == nil || w.svc == nil || w.executor == nil {
		return nil
	}
	if w.opKeyGate != nil && w.opKeyGate.ShouldShed() {
		return nil
	}
	q := db.New(w.svc.pool)
	rows, err := q.ListBookedOperationLeasesForNode(ctx, db.ListBookedOperationLeasesForNodeParams{
		NodeID:   w.nodeID,
		RowLimit: 32,
	})
	if err != nil {
		return fmt.Errorf("operation lease process booked node=%s: %w", w.nodeID, err)
	}
	metrics.OpBookedQueueDepth.Set(float64(len(rows)))
	for _, row := range rows {
		opID := uuid.UUID(row.OpID.Bytes)
		if err := w.ExecuteOp(ctx, opID, w.executor); err != nil {
			if errors.Is(err, ErrStaleFencingEpoch) {
				slog.Warn("operation lease stale fencing", "op_id", opID)
				continue
			}
			slog.Warn("operation lease execute failed", "op_id", opID, "err", err)
		}
	}
	return nil
}

func (w *OperationLeaseWorker) Start(ctx context.Context) {
	if w == nil || w.svc == nil || w.svc.pool == nil {
		return
	}
	slog.Info("operation lease worker starting",
		"node_id", w.nodeID,
		"role", w.role,
		"region", w.region,
		"timeout_sec", w.timeoutSec,
		"janitor_period", w.janitorPeriod,
	)
	pollTicker := time.NewTicker(w.pollInterval)
	defer pollTicker.Stop()
	janitorTicker := time.NewTicker(w.janitorPeriod)
	defer janitorTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-pollTicker.C:
			if err := w.ProcessBooked(ctx); err != nil && ctx.Err() == nil {
				slog.Error("operation lease poll failed", "node_id", w.nodeID, "err", err)
			}
		case <-janitorTicker.C:
			if _, err := w.RunJanitor(ctx); err != nil && ctx.Err() == nil {
				slog.Error("operation lease janitor failed", "node_id", w.nodeID, "err", err)
			}
		}
	}
}

func LeaseOpID(lease db.OperationLease) uuid.UUID {
	if lease.OpID.Valid {
		return uuid.UUID(lease.OpID.Bytes)
	}
	return uuid.Nil
}

func LeaseFactorU(lease db.OperationLease) uuid.UUID {
	if lease.FactorU.Valid {
		return uuid.UUID(lease.FactorU.Bytes)
	}
	return uuid.Nil
}

func LeaseDeadline(lease db.OperationLease) time.Time {
	if lease.DeadlineAt.Valid {
		return lease.DeadlineAt.Time
	}
	return time.Time{}
}
