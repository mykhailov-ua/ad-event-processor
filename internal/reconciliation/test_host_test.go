package reconciliation

import (
	"context"
	"errors"
	"testing"
	"time"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/database"
	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type testHost struct {
	pool            *pgxpool.Pool
	settlementPool  *pgxpool.Pool
	paymentPool     PaymentQueryer
	redisShards     []redis.UniversalClient
	sharder         domain.Sharder
	cfg             *config.Config
	alerter         Alerter
	clickhouseQuery *database.ClickHouseQuery
	brokerDeltas    BrokerPendingDeltaReader
}

func newTestHost(t *testing.T, pool *pgxpool.Pool, redisShards []redis.UniversalClient, cfg *config.Config) *testHost {
	t.Helper()
	if cfg == nil {
		cfg = &config.Config{}
	}
	shardCount := len(redisShards)
	if shardCount == 0 {
		shardCount = 1
	}
	h := &testHost{
		pool:        pool,
		redisShards: redisShards,
		sharder:     domain.NewStaticSlotSharder(shardCount),
		cfg:         cfg,
	}
	h.settlementPool = pool
	h.paymentPool = pool
	return h
}

func (h *testHost) Pool() *pgxpool.Pool { return h.pool }

func (h *testHost) SettlementPool() *pgxpool.Pool {
	if h.settlementPool != nil {
		return h.settlementPool
	}
	return h.pool
}

func (h *testHost) PaymentQueryPool() PaymentQueryer {
	if h.paymentPool != nil {
		return h.paymentPool
	}
	return h.pool
}

func (h *testHost) RedisShards() []redis.UniversalClient { return h.redisShards }

func (h *testHost) Sharder() domain.Sharder { return h.sharder }

func (h *testHost) Config() *config.Config { return h.cfg }

func (h *testHost) WithPostgresLow(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

func (h *testHost) ClickHouseQuery() *database.ClickHouseQuery { return h.clickhouseQuery }

func (h *testHost) RedisClientForCampaign(campaignID uuid.UUID) redis.UniversalClient {
	shard := h.sharder.GetShard(campaignID)
	if shard < 0 || shard >= len(h.redisShards) {
		return nil
	}
	return h.redisShards[shard]
}

func (h *testHost) AuditLog(ctx context.Context, q db.Querier, adminID uuid.UUID, action, targetType string, targetID *uuid.UUID, changes, metadata any) {
}

func (h *testHost) Alerter() Alerter { return h.alerter }

func (h *testHost) ForceRefillCampaignFromPG(ctx context.Context, campaignID uuid.UUID, currentSpend int64) error {
	var budgetLimit int64
	err := h.pool.QueryRow(ctx, `SELECT budget_limit FROM campaigns WHERE id = $1`, domain.ToUUID(campaignID)).Scan(&budgetLimit)
	if err != nil {
		return err
	}
	remaining := budgetLimit - currentSpend
	if remaining < 0 {
		remaining = 0
	}
	redisClient := h.RedisClientForCampaign(campaignID)
	if redisClient == nil {
		return errors.New("no redis shard")
	}
	return redisClient.Set(ctx, domain.BudgetCampaignKey(campaignID), remaining, 0).Err()
}

func (h *testHost) BrokerDeltas() BrokerPendingDeltaReader { return h.brokerDeltas }

func (h *testHost) InvalidServiceFilterErr() error { return ErrInvalidServiceFilter }

func (h *testHost) RunStuckDrainCheck(ctx context.Context) {}

type stubAlerter struct {
	unresolved []struct {
		runID      int64
		unresolved int
		totalDelta int64
		period     string
	}
	discrepancy []struct {
		runID         int64
		discrepancies int
		totalDelta    int64
		period        string
	}
}

func (s *stubAlerter) AlertReconDiscrepancy(ctx context.Context, runID int64, discrepancies int, totalDelta int64, period string) {
	s.discrepancy = append(s.discrepancy, struct {
		runID         int64
		discrepancies int
		totalDelta    int64
		period        string
	}{runID, discrepancies, totalDelta, period})
}

func (s *stubAlerter) AlertReconDiscrepancyUnresolved(ctx context.Context, runID int64, unresolved int, totalDelta int64, period string, oldest time.Time) {
	s.unresolved = append(s.unresolved, struct {
		runID      int64
		unresolved int
		totalDelta int64
		period     string
	}{runID, unresolved, totalDelta, period})
}

func applyPendingReconciliationAdjusts(ctx context.Context, host Host, pool *pgxpool.Pool) error {
	rows, err := pool.Query(ctx, `
		SELECT id, payload FROM outbox_events
		WHERE event_type = 'RECONCILIATION_ADJUST' AND status = 'PENDING'
		ORDER BY id`)
	if err != nil {
		return err
	}
	defer rows.Close()

	applier := NewAdjustApplier(host)
	for rows.Next() {
		var id int64
		var payload []byte
		if err := rows.Scan(&id, &payload); err != nil {
			return err
		}
		if err := applier.Apply(ctx, id, payload); err != nil {
			return err
		}
		_, err = pool.Exec(ctx, `UPDATE outbox_events SET status = 'PROCESSED', processed_at = NOW() WHERE id = $1`, id)
		if err != nil {
			return err
		}
	}
	return rows.Err()
}
