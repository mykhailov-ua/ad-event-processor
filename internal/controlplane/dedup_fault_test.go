package controlplane

import (
	"context"
	"encoding/json"
	"strconv"
	"testing"

	"ad-event-processor/pkg/faultproof"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/database"
	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestFault_DedupMultiRegionDuplicate(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
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
		INSERT INTO customers (id, name, balance, currency) VALUES ($1, 'dedup-relay', 0, 'USD')`,
		domain.ToUUID(customerID))
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `
		INSERT INTO campaigns (id, name, budget_limit, current_spend, status, customer_id, pacing_mode, timezone, freq_window)
		VALUES ($1, 'dedup-camp', 8000000, 0, 'ACTIVE', $2, 'ASAP', 'UTC', 86400)`,
		domain.ToUUID(campaignID), domain.ToUUID(customerID))
	require.NoError(t, err)

	payload, err := json.Marshal(CampaignPayload{
		CampaignID:  campaignID.String(),
		BudgetLimit: 8_000_000,
	})
	require.NoError(t, err)

	var eventID int64
	err = pool.QueryRow(ctx, `
		INSERT INTO outbox_events (event_type, payload) VALUES ('CREATE_CAMPAIGN', $1) RETURNING id`, payload).Scan(&eventID)
	require.NoError(t, err)

	cfg := &config.Config{MultiRegionEnabled: true, RegionCode: 1}
	svc := newBareService(t, pool, []redis.UniversalClient{rdb}, cfg)
	relay := NewRegionOutboxRelay(svc)

	for range 3 {
		_, _ = pool.Exec(ctx, `
			UPDATE outbox_region_delivery
			SET status = 'PENDING', processing_started_at = NULL, delivered_at = NULL
			WHERE region_code = 1 AND outbox_event_id = $1`, eventID)
		require.NoError(t, relay.ProcessPending(ctx))
	}

	budgetKey := "budget:campaign:" + campaignID.String()
	val, err := rdb.Get(ctx, budgetKey).Int64()
	require.NoError(t, err)
	require.Equal(t, int64(8_000_000), val)

	var proposalCount int
	require.NoError(t, pool.QueryRow(ctx, `SELECT COUNT(*) FROM dedup_key_proposals`).Scan(&proposalCount))
	require.Equal(t, 1, proposalCount)

	opID := RelayDeliveryOpID(1, eventID)
	var leaseState string
	require.NoError(t, pool.QueryRow(ctx, `
		SELECT lease_state FROM operation_leases WHERE op_id = $1`, domain.ToUUID(opID)).Scan(&leaseState))
	require.Equal(t, string(LeaseStateCompleted), leaseState)

	var idemCount int
	require.NoError(t, pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM region_apply_idempotency WHERE region_code = 1 AND outbox_event_id = $1`, eventID).Scan(&idemCount))
	require.Equal(t, 1, idemCount)

	faultproof.Log(t, "dedup_multi_region_duplicate", map[string]string{
		"subsystem":     "region_outbox_relay",
		"region_code":   "1",
		"event_id":      strconv.FormatInt(eventID, 10),
		"redis_budget":  strconv.FormatInt(val, 10),
		"proposal_rows": strconv.Itoa(proposalCount),
		"deliveries":    "3",
		"lease_state":   leaseState,
		"baseline_ok":   "true",
	})
}
