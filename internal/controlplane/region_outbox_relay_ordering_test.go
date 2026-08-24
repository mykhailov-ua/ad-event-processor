package controlplane

import (
	"context"
	"encoding/json"
	"testing"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/database"
	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegionOutboxRelay_strictOrderHaltsOnFailure(t *testing.T) {
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

	badCreatePayload := []byte("{")
	updatePayload := []byte(`{"settings":{"iso04_marker":"held_out"}}`)

	var createEventID int64
	err = pool.QueryRow(ctx, `
		INSERT INTO outbox_events (event_type, payload, created_at)
		VALUES ('CREATE_CAMPAIGN', $1, NOW() - INTERVAL '2 seconds') RETURNING id`, badCreatePayload).Scan(&createEventID)
	require.NoError(t, err)

	var updateEventID int64
	err = pool.QueryRow(ctx, `
		INSERT INTO outbox_events (event_type, payload, created_at)
		VALUES ('UPDATE_SETTINGS', $1, NOW() - INTERVAL '1 second') RETURNING id`, updatePayload).Scan(&updateEventID)
	require.NoError(t, err)

	cfg := &config.Config{
		RegionCode: 1,
	}
	svc := newBareService(t, pool, []redis.UniversalClient{rdb}, cfg)
	relay := NewRegionOutboxRelay(svc)

	_, err = relay.ProcessPendingWithCount(ctx, 10)
	require.Error(t, err, "harness=region_outbox_relay_in_process: CREATE must fail and halt batch")

	var createStatus, updateStatus string
	require.NoError(t, pool.QueryRow(ctx, `
		SELECT status FROM outbox_region_delivery WHERE outbox_event_id = $1 AND region_code = 1`,
		createEventID).Scan(&createStatus))
	require.NoError(t, pool.QueryRow(ctx, `
		SELECT status FROM outbox_region_delivery WHERE outbox_event_id = $1 AND region_code = 1`,
		updateEventID).Scan(&updateStatus))
	assert.Equal(t, "PENDING", createStatus)
	assert.Equal(t, "PENDING", updateStatus, "UPDATE must not run while CREATE is still pending")

	version, err := rdb.Get(ctx, "config:version").Int64()
	if err == nil {
		assert.NotEqual(t, updateEventID, version, "UPDATE_SETTINGS must not apply when batch halted on CREATE")
	} else {
		require.ErrorIs(t, err, redis.Nil)
	}

	var idemCount int
	require.NoError(t, pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM region_apply_idempotency WHERE region_code = 1 AND outbox_event_id = $1`,
		updateEventID).Scan(&idemCount))
	assert.Equal(t, 0, idemCount)
}

func TestRegionOutboxRelay_strictOrderSucceedsAfterCreateFixed(t *testing.T) {
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
		INSERT INTO customers (id, name, balance, currency) VALUES ($1, 'iso04-order', 0, 'USD')`,
		domain.ToUUID(customerID))
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `
		INSERT INTO campaigns (id, name, budget_limit, current_spend, status, customer_id, pacing_mode, timezone, freq_window)
		VALUES ($1, 'iso04-order', 3000000, 0, 'ACTIVE', $2, 'ASAP', 'UTC', 86400)`,
		domain.ToUUID(campaignID), domain.ToUUID(customerID))
	require.NoError(t, err)

	createPayload, err := json.Marshal(CampaignPayload{
		CampaignID:  campaignID.String(),
		BudgetLimit: 3_000_000,
	})
	require.NoError(t, err)
	updatePayload := []byte(`{"settings":{"iso04_marker":"after_create"}}`)

	var createEventID int64
	err = pool.QueryRow(ctx, `
		INSERT INTO outbox_events (event_type, payload, created_at)
		VALUES ('CREATE_CAMPAIGN', $1, NOW() - INTERVAL '2 seconds') RETURNING id`, createPayload).Scan(&createEventID)
	require.NoError(t, err)

	var updateEventID int64
	err = pool.QueryRow(ctx, `
		INSERT INTO outbox_events (event_type, payload, created_at)
		VALUES ('UPDATE_SETTINGS', $1, NOW() - INTERVAL '1 second') RETURNING id`, updatePayload).Scan(&updateEventID)
	require.NoError(t, err)

	cfg := &config.Config{
		RegionCode: 1,
	}
	svc := newBareService(t, pool, []redis.UniversalClient{rdb}, cfg)
	relay := NewRegionOutboxRelay(svc)

	delivered, err := relay.ProcessPendingWithCount(ctx, 10)
	require.NoError(t, err)
	assert.Equal(t, 2, delivered)

	version, err := rdb.Get(ctx, "config:version").Int64()
	require.NoError(t, err)
	assert.Equal(t, updateEventID, version)

	budgetKey := "budget:campaign:" + campaignID.String()
	budget, err := rdb.Get(ctx, budgetKey).Int64()
	require.NoError(t, err)
	assert.Equal(t, int64(3_000_000), budget)
}
