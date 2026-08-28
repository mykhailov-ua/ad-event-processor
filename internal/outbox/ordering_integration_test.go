package outbox_test

import (
	"context"
	"encoding/json"
	"testing"

	"ad-event-processor/internal/database"
	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/outbox"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorker_strictOrderHaltsOnFailure(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}

	ctx := context.Background()
	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()
	redisClient, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	badCreatePayload := []byte("{")
	updatePayload := []byte(`{"settings":{"iso05_marker":"held_out"}}`)

	var createEventID int64
	err := pool.QueryRow(ctx, `
		INSERT INTO outbox_events (event_type, payload, created_at)
		VALUES ('CREATE_CAMPAIGN', $1, NOW() - INTERVAL '2 seconds') RETURNING id`, badCreatePayload).Scan(&createEventID)
	require.NoError(t, err)

	var updateEventID int64
	err = pool.QueryRow(ctx, `
		INSERT INTO outbox_events (event_type, payload, created_at)
		VALUES ('UPDATE_SETTINGS', $1, NOW() - INTERVAL '1 second') RETURNING id`, updatePayload).Scan(&updateEventID)
	require.NoError(t, err)

	worker := newWorker(t, pool, []redis.UniversalClient{redisClient}, nil)

	processed, err := worker.ProcessOutboxWithCount(ctx, 10)
	require.Error(t, err, "harness=management_outbox_in_process: CREATE must fail and halt batch")
	assert.Equal(t, 0, processed)

	var createStatus, updateStatus string
	require.NoError(t, pool.QueryRow(ctx, `
		SELECT status FROM outbox_events WHERE id = $1`, createEventID).Scan(&createStatus))
	require.NoError(t, pool.QueryRow(ctx, `
		SELECT status FROM outbox_events WHERE id = $1`, updateEventID).Scan(&updateStatus))
	assert.Equal(t, "PENDING", createStatus)
	assert.Equal(t, "PENDING", updateStatus, "UPDATE must not run while CREATE is still pending")

	version, err := redisClient.Get(ctx, "config:version").Int64()
	if err == nil {
		assert.NotEqual(t, updateEventID, version, "UPDATE_SETTINGS must not apply when batch halted on CREATE")
	} else {
		require.ErrorIs(t, err, redis.Nil)
	}
}

func TestWorker_strictOrderSucceedsInSequence(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}

	ctx := context.Background()
	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()
	redisClient, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	customerID := uuid.New()
	campaignID := uuid.New()
	_, err := pool.Exec(ctx, `
		INSERT INTO customers (id, name, balance, currency) VALUES ($1, 'iso05-order', 0, 'USD')`,
		domain.ToUUID(customerID))
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `
		INSERT INTO campaigns (id, name, budget_limit, current_spend, status, customer_id, pacing_mode, timezone, freq_window)
		VALUES ($1, 'iso05-order', 3000000, 0, 'ACTIVE', $2, 'ASAP', 'UTC', 86400)`,
		domain.ToUUID(campaignID), domain.ToUUID(customerID))
	require.NoError(t, err)

	createPayload, err := json.Marshal(outbox.CampaignPayload{
		CampaignID:  campaignID.String(),
		BudgetLimit: 3_000_000,
	})
	require.NoError(t, err)
	updatePayload := []byte(`{"settings":{"iso05_marker":"after_create"}}`)

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

	worker := newWorker(t, pool, []redis.UniversalClient{redisClient}, nil)

	processed, err := worker.ProcessOutboxWithCount(ctx, 10)
	require.NoError(t, err)
	assert.Equal(t, 2, processed)

	version, err := redisClient.Get(ctx, "config:version").Int64()
	require.NoError(t, err)
	assert.Equal(t, updateEventID, version)

	budgetKey := "budget:campaign:" + campaignID.String()
	budget, err := redisClient.Get(ctx, budgetKey).Int64()
	require.NoError(t, err)
	assert.Equal(t, int64(3_000_000), budget)
}
