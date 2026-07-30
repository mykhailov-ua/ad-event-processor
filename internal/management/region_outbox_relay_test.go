package management

import (
	"espx/pkg/faultproof"

	"context"
	"encoding/json"
	"strconv"
	"testing"
	"time"

	"espx/internal/config"
	"espx/internal/database"
	"espx/internal/dedup"
	"espx/internal/ingestion"
	db "espx/internal/ingestion/sqlc"
	"espx/pkg/dedupkey"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestRegionCellIsolation(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test")
	}

	ctx := context.Background()
	rdbA, cleanupA := database.SetupTestRedis(t)
	defer cleanupA()
	rdbB, cleanupB := database.SetupTestRedis(t)
	defer cleanupB()

	key := "ingress:day:01:" + uuid.New().String() + ":20260720"
	require.NoError(t, rdbA.Set(ctx, key, 42, time.Hour).Err())

	valA, err := rdbA.Get(ctx, key).Int64()
	require.NoError(t, err)
	require.Equal(t, int64(42), valA)

	_, err = rdbB.Get(ctx, key).Result()
	require.ErrorIs(t, err, redis.Nil)

	faultproof.Log(t, "region_cell_isolation", map[string]string{
		"subsystem":  "multi_region",
		"cell_a_key": key,
		"cell_b_hit": "false",
	})
}

func TestRegionOutboxRelay(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test")
	}

	ctx := context.Background()
	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()
	rdb, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	_, err := pool.Exec(ctx, `
		INSERT INTO regions (code, name, active) VALUES (1, 'us-east', TRUE), (2, 'eu-west', TRUE)
		ON CONFLICT (code) DO NOTHING`)
	require.NoError(t, err)

	customerID := uuid.New()
	campaignID := uuid.New()
	_, err = pool.Exec(ctx, `
		INSERT INTO customers (id, name, balance, currency) VALUES ($1, 'region-relay', 0, 'USD')`,
		ingestion.ToUUID(customerID))
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `
		INSERT INTO campaigns (id, name, budget_limit, current_spend, status, customer_id, pacing_mode, timezone, freq_window)
		VALUES ($1, 'relay-camp', 5000000, 0, 'ACTIVE', $2, 'ASAP', 'UTC', 86400)`,
		ingestion.ToUUID(campaignID), ingestion.ToUUID(customerID))
	require.NoError(t, err)

	payload, err := json.Marshal(CampaignPayload{
		CampaignID:  campaignID.String(),
		BudgetLimit: 5_000_000,
	})
	require.NoError(t, err)

	var eventID int64
	err = pool.QueryRow(ctx, `
		INSERT INTO outbox_events (event_type, payload) VALUES ('CREATE_CAMPAIGN', $1) RETURNING id`, payload).Scan(&eventID)
	require.NoError(t, err)

	var deliveryStatus string
	err = pool.QueryRow(ctx, `
		SELECT status FROM outbox_region_delivery WHERE outbox_event_id = $1 AND region_code = 1`, eventID).Scan(&deliveryStatus)
	require.NoError(t, err)
	require.Equal(t, "PENDING", deliveryStatus)

	cfg := &config.Config{
		MultiRegionEnabled: true,
		RegionCode:         1,
	}
	svc := newBareService(t, pool, []redis.UniversalClient{rdb}, cfg)
	relay := NewRegionOutboxRelay(svc)
	require.NoError(t, relay.ProcessPending(ctx))

	err = pool.QueryRow(ctx, `
		SELECT status FROM outbox_region_delivery WHERE outbox_event_id = $1 AND region_code = 1`, eventID).Scan(&deliveryStatus)
	require.NoError(t, err)
	require.Equal(t, "DELIVERED", deliveryStatus)

	budgetKey := "budget:campaign:" + campaignID.String()
	val, err := rdb.Get(ctx, budgetKey).Int64()
	require.NoError(t, err)
	require.Equal(t, int64(5_000_000), val)

	var idemCount int
	require.NoError(t, pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM region_apply_idempotency WHERE region_code = 1 AND outbox_event_id = $1`, eventID).Scan(&idemCount))
	require.Equal(t, 1, idemCount)

	require.NoError(t, relay.ProcessPending(ctx))
	val2, err := rdb.Get(ctx, budgetKey).Int64()
	require.NoError(t, err)
	require.Equal(t, val, val2)

	faultproof.Log(t, "region_outbox_relay", map[string]string{
		"subsystem":    "region_outbox_relay",
		"region_code":  "1",
		"event_id":     strconv.FormatInt(eventID, 10),
		"redis_budget": strconv.FormatInt(val, 10),
		"delivered":    "true",
	})
}

func TestRegionOutboxRelay_WithOperationLease(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test")
	}

	ctx := context.Background()
	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()
	rdb, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	_, err := pool.Exec(ctx, `
		INSERT INTO regions (code, name, active) VALUES (1, 'us-east', TRUE)
		ON CONFLICT (code) DO NOTHING`)
	require.NoError(t, err)

	customerID := uuid.New()
	campaignID := uuid.New()
	_, err = pool.Exec(ctx, `
		INSERT INTO customers (id, name, balance, currency) VALUES ($1, 'lease-relay', 0, 'USD')`,
		ingestion.ToUUID(customerID))
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `
		INSERT INTO campaigns (id, name, budget_limit, current_spend, status, customer_id, pacing_mode, timezone, freq_window)
		VALUES ($1, 'lease-relay-camp', 6000000, 0, 'ACTIVE', $2, 'ASAP', 'UTC', 86400)`,
		ingestion.ToUUID(campaignID), ingestion.ToUUID(customerID))
	require.NoError(t, err)

	payload, err := json.Marshal(CampaignPayload{
		CampaignID:  campaignID.String(),
		BudgetLimit: 6_000_000,
	})
	require.NoError(t, err)

	var eventID int64
	err = pool.QueryRow(ctx, `
		INSERT INTO outbox_events (event_type, payload) VALUES ('CREATE_CAMPAIGN', $1) RETURNING id`, payload).Scan(&eventID)
	require.NoError(t, err)

	cfg := &config.Config{
		MultiRegionEnabled: true,
		RegionCode:         1,
		NodeID:             "relay-lease-1",
		OpLeaseTimeoutSec:  30,
	}
	svc := newBareService(t, pool, []redis.UniversalClient{rdb}, cfg)
	relay := NewRegionOutboxRelay(svc)
	leaseWorker := NewOperationLeaseWorker(svc)

	opID := uuid.New()
	factorU := dedupkey.FactorU(dedupkey.CanonicalRelayPayload(eventID, "CREATE_CAMPAIGN", payload))
	scope := dedup.NewAdapter(pool, 1, 1).RegionScope(dedupkey.RelaySourceID(1), eventID, eventID)
	_, err = leaseWorker.Book(ctx, OperationLeaseBookRequest{
		OpID:         opID,
		RegionCode:   1,
		Role:         "management",
		ReplicaSetID: uuid.New(),
		Attempt:      1,
		FactorU:      factorU,
		Scope:        scope,
		ReplicaNodes: []string{cfg.NodeID},
	})
	require.NoError(t, err)

	err = leaseWorker.ExecuteOp(ctx, opID, func(ctx context.Context, _ db.OperationLease, _ dedup.ClaimResult) error {
		return relay.ProcessPending(ctx)
	})
	require.NoError(t, err)

	var deliveryStatus string
	err = pool.QueryRow(ctx, `
		SELECT status FROM outbox_region_delivery WHERE outbox_event_id = $1 AND region_code = 1`, eventID).Scan(&deliveryStatus)
	require.NoError(t, err)
	require.Equal(t, "DELIVERED", deliveryStatus)

	lease, err := db.New(pool).GetOperationLease(ctx, ingestion.ToUUID(opID))
	require.NoError(t, err)
	require.Equal(t, string(LeaseStateCompleted), lease.LeaseState)

	budgetKey := "budget:campaign:" + campaignID.String()
	val, err := rdb.Get(ctx, budgetKey).Int64()
	require.NoError(t, err)
	require.Equal(t, int64(6_000_000), val)

	faultproof.Log(t, "region_outbox_relay_lease", map[string]string{
		"subsystem":    "region_outbox_relay",
		"lease_path":   "true",
		"region_code":  "1",
		"event_id":     strconv.FormatInt(eventID, 10),
		"redis_budget": strconv.FormatInt(val, 10),
		"lease_state":  lease.LeaseState,
	})
}
