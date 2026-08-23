package fraud

import (
	"context"
	"testing"

	"github.com/bidshard/ad-event-processor/internal/database"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOutboxBackpressure_enforcementExcluded_holdout(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: outbox backpressure enforcement exclusion (run make test-integration)")
	}

	pool, cleanup := database.SetupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	for range 10 {
		_, err := pool.Exec(ctx, `
			INSERT INTO outbox_events (event_type, payload, status)
			VALUES ('ML_SCORE_BOOST', '{"action":"boost"}', 'PENDING')`)
		require.NoError(t, err)
	}

	detector := NewDetector(
		stubFinder{ips: []SuspiciousIP{fraudBoostCandidate("203.0.113.60")}},
		NewIdempotencyStore(pool),
		&countingManagement{},
		pool,
		DetectorConfig{OutboxPendingLimit: 5},
	)

	result, err := detector.Run(ctx)
	require.NoError(t, err)
	assert.False(t, result.Backlogged)
	assert.Equal(t, 1, result.Enqueued)
}

func TestOutboxBackpressure_pacingTriggers_holdout(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: outbox backpressure pacing trigger (run make test-integration)")
	}

	pool, cleanup := database.SetupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	for range 5 {
		_, err := pool.Exec(ctx, `
			INSERT INTO outbox_events (event_type, payload, status)
			VALUES ('UPDATE_CAMPAIGN_PACING', '{"campaign_id":"`+uuid.New().String()+`","pacing_mode":"even"}', 'PENDING')`)
		require.NoError(t, err)
	}

	detector := NewDetector(
		stubFinder{ips: []SuspiciousIP{fraudBoostCandidate("203.0.113.61")}},
		NewIdempotencyStore(pool),
		&countingManagement{},
		pool,
		DetectorConfig{OutboxPendingLimit: 3},
	)

	result, err := detector.Run(ctx)
	require.ErrorIs(t, err, ErrOutboxBackpressure)
	assert.True(t, result.Backlogged)
}
