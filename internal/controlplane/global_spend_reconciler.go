package controlplane

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/metrics"
	"ad-event-processor/pkg/dedupkey"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

const (
	defaultGlobalSpendBatchMin       = 100
	defaultGlobalSpendMaxConcurrency = 8
	defaultGlobalSpendFlushInterval  = 500 * time.Millisecond
	globalSpendIdempotencyPrefix     = "global_spend:"
)

var ErrSpendBatchTooSmall = errors.New("spend sync batch below minimum txn count")

const globalSpendCommitScript = `
local marker = KEYS[2]
if redis.call("EXISTS", marker) == 1 then
	return 0
end
local remaining = redis.call("INCRBY", KEYS[1], -tonumber(ARGV[1]))
if tonumber(remaining) <= 0 then
 redis.call("DEL", KEYS[1])
end
redis.call("SET", marker, "1", "EX", tonumber(ARGV[2]))
return remaining
`

const globalSpendRedisMarkerTTL = 7 * 24 * time.Hour

func globalSpendRedisMarkerKey(batchDedupKey string, campaignID uuid.UUID) string {
	return fmt.Sprintf("global_spend:redis_applied:%s:%s", batchDedupKey, campaignID.String())
}

type GlobalSpendReconciler struct {
	pool           *pgxpool.Pool
	redisShards    []redis.UniversalClient
	sharder        domain.Sharder
	campaignRepo   *domain.CampaignRepo
	minBatchSize   int
	maxConcurrency int

	mu      sync.Mutex
	pending []dedupkey.SpendSyncTxn
}

type GlobalSpendReconcilerConfig struct {
	MinBatchSize   int
	MaxConcurrency int
}

func NewGlobalSpendReconciler(
	pool *pgxpool.Pool,
	redisShards []redis.UniversalClient,
	sharder domain.Sharder,
	cfg GlobalSpendReconcilerConfig,
) *GlobalSpendReconciler {
	if cfg.MinBatchSize <= 0 {
		cfg.MinBatchSize = defaultGlobalSpendBatchMin
	}
	if cfg.MaxConcurrency <= 0 {
		cfg.MaxConcurrency = defaultGlobalSpendMaxConcurrency
	}
	var repo *domain.CampaignRepo
	if pool != nil {
		repo = domain.NewCampaignRepoWithDB(pool, db.New(pool))
	}
	return &GlobalSpendReconciler{
		pool:           pool,
		redisShards:    redisShards,
		sharder:        sharder,
		campaignRepo:   repo,
		minBatchSize:   cfg.MinBatchSize,
		maxConcurrency: cfg.MaxConcurrency,
	}
}

func (r *GlobalSpendReconciler) MinBatchSize() int {
	if r == nil {
		return defaultGlobalSpendBatchMin
	}
	return r.minBatchSize
}

func (r *GlobalSpendReconciler) Enqueue(txns []dedupkey.SpendSyncTxn) {
	if r == nil || len(txns) == 0 {
		return
	}
	r.mu.Lock()
	r.pending = append(r.pending, txns...)
	r.mu.Unlock()
}

func (r *GlobalSpendReconciler) PendingCount() int {
	if r == nil {
		return 0
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.pending)
}

func (r *GlobalSpendReconciler) FlushPending(ctx context.Context, batchDedupKey string) error {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	if len(r.pending) < r.minBatchSize {
		r.mu.Unlock()
		return nil
	}
	txns := append([]dedupkey.SpendSyncTxn(nil), r.pending...)
	r.pending = r.pending[:0]
	r.mu.Unlock()
	return r.ApplyBatch(ctx, batchDedupKey, txns)
}

func (r *GlobalSpendReconciler) ApplyBatch(ctx context.Context, batchDedupKey string, txns []dedupkey.SpendSyncTxn) error {
	if r == nil {
		return nil
	}
	if r.pool == nil || r.campaignRepo == nil {
		return fmt.Errorf("global spend reconciler: postgres unavailable")
	}
	if len(txns) < r.minBatchSize {
		return fmt.Errorf("global spend batch dedup=%s: %w (%d < %d)", batchDedupKey, ErrSpendBatchTooSmall, len(txns), r.minBatchSize)
	}

	start := time.Now()
	items := make([]domain.SpendFlushItem, 0, len(txns))
	redisDeltas := make(map[uuid.UUID]int64, len(txns))
	for _, txn := range txns {
		idemID := globalSpendIdempotencyPrefix + batchDedupKey + ":" + txn.TxnID
		items = append(items, domain.SpendFlushItem{
			CampaignID:  txn.CampaignID,
			AmountMicro: txn.AmountMicro,
			TxID:        idemID,
			StrictFlush: true,
		})
		redisDeltas[txn.CampaignID] += txn.AmountMicro
	}

	pgApplied, err := r.globalSpendPgApplied(ctx, items[0].TxID)
	if err != nil {
		return fmt.Errorf("global spend batch dedup=%s pg probe: %w", batchDedupKey, err)
	}
	if !pgApplied {
		for startIdx := 0; startIdx < len(items); startIdx += domain.MaxLedgerBatchSize() {
			endIdx := startIdx + domain.MaxLedgerBatchSize()
			if endIdx > len(items) {
				endIdx = len(items)
			}
			chunk := items[startIdx:endIdx]
			outcomes, err := r.campaignRepo.UpdateSpendBatch(ctx, chunk)
			if err != nil {
				return fmt.Errorf("global spend batch dedup=%s: %w", batchDedupKey, err)
			}
			for _, outcome := range outcomes {
				if outcome.Err != nil && !errors.Is(outcome.Err, domain.ErrInsufficientCustomerBalance) {
					return fmt.Errorf("global spend batch dedup=%s campaign=%s: %w", batchDedupKey, outcome.CampaignID, outcome.Err)
				}
			}
		}
	}

	if err := r.commitRedisBudget(ctx, batchDedupKey, redisDeltas); err != nil {
		metrics.GlobalSpendFlushErrorsTotal.Inc()
		return fmt.Errorf("global spend batch dedup=%s redis commit: %w", batchDedupKey, err)
	}

	elapsed := time.Since(start).Seconds()
	metrics.GlobalSpendBatchesTotal.Inc()
	metrics.GlobalSpendTxnsTotal.Add(float64(len(txns)))
	metrics.GlobalSpendBatchSize.Observe(float64(len(txns)))
	metrics.GlobalSpendApplyLatency.Observe(elapsed)
	return nil
}

func (r *GlobalSpendReconciler) globalSpendPgApplied(ctx context.Context, probeTxID string) (bool, error) {
	if r == nil || r.pool == nil || probeTxID == "" {
		return false, nil
	}
	var exists bool
	err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM sync_idempotency WHERE id = $1)`, probeTxID).Scan(&exists)
	return exists, err
}

func (r *GlobalSpendReconciler) commitRedisBudget(ctx context.Context, batchDedupKey string, deltas map[uuid.UUID]int64) error {
	if len(r.redisShards) == 0 {
		return nil
	}
	ttlSec := int64(globalSpendRedisMarkerTTL / time.Second)
	sem := make(chan struct{}, r.maxConcurrency)
	var wg sync.WaitGroup
	var firstErr error
	var errMu sync.Mutex

	for campID, amount := range deltas {
		if amount <= 0 {
			continue
		}
		shardIdx := r.shardIndex(campID)
		if shardIdx < 0 || shardIdx >= len(r.redisShards) {
			continue
		}
		redisClient := r.redisShards[shardIdx]
		budgetKey := domain.BudgetCampaignKey(campID)
		markerKey := globalSpendRedisMarkerKey(batchDedupKey, campID)
		wg.Add(1)
		sem <- struct{}{}
		go func(redisClient redis.UniversalClient, budgetKey, markerKey string, amount int64) {
			defer wg.Done()
			defer func() { <-sem }()
			_, err := redisClient.Eval(ctx, globalSpendCommitScript, []string{budgetKey, markerKey}, amount, ttlSec).Result()
			if err != nil {
				errMu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				errMu.Unlock()
			}
		}(redisClient, budgetKey, markerKey, amount)
	}
	wg.Wait()
	return firstErr
}

func (r *GlobalSpendReconciler) shardIndex(campaignID uuid.UUID) int {
	if r.sharder == nil {
		return 0
	}
	return int(r.sharder.GetShard(campaignID))
}

func (s *Service) applyRegionSpendSyncBatch(ctx context.Context, batchDedupKey string, payload []byte) error {
	if s == nil || s.globalSpend == nil {
		return nil
	}
	if !dedupkey.IsSpendSyncPayload(payload) {
		return nil
	}
	txns, err := dedupkey.DecodeSpendSyncPayload(payload)
	if err != nil {
		return err
	}
	return s.globalSpend.ApplyBatch(ctx, batchDedupKey, txns)
}

func (r *GlobalSpendReconciler) StartFlushWorker(ctx context.Context, interval time.Duration) {
	if r == nil {
		return
	}
	if interval <= 0 {
		interval = defaultGlobalSpendFlushInterval
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			flushKey := uuid.New().String()
			if err := r.FlushPending(ctx, flushKey); err != nil && !errors.Is(err, ErrSpendBatchTooSmall) {
				metrics.GlobalSpendFlushErrorsTotal.Inc()
			}
		}
	}
}
