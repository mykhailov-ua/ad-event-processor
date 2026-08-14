package controlplane

import (
	"context"
	"testing"

	"github.com/bidshard/ad-event-processor/internal/testutil"
	"github.com/bidshard/ad-event-processor/pkg/faultproof"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestFault_SettlementReplay(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	cfg := testutil.DefaultPostgresConfig()
	cfg.MigrationDirs = []string{testutil.AdsMigrationsDir()}
	pool, cleanup := testutil.SetupPostgres(t, cfg)
	defer cleanup()

	ctx := context.Background()
	eventID := "evt-" + uuid.New().String()
	campaignID := uuid.New()

	for i := range 3 {
		id := eventID
		if i > 0 {
			id = eventID + "-retry-" + uuid.New().String()[:8]
		}
		_, err := pool.Exec(ctx, `
			INSERT INTO sync_idempotency (id, event_id, campaign_id)
			VALUES ($1, $2, $3)
			ON CONFLICT (event_id, campaign_id) WHERE event_id IS NOT NULL AND campaign_id IS NOT NULL DO NOTHING`,
			id, eventID, campaignID)
		require.NoError(t, err)
	}

	var count int
	require.NoError(t, pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM sync_idempotency WHERE event_id = $1 AND campaign_id = $2`,
		eventID, campaignID).Scan(&count))
	require.Equal(t, 1, count)

	faultproof.Log(t, "settlement_replay", map[string]string{
		"fault":         "settlement_replay",
		"proposal_rows": "1",
		"event_id":      eventID,
		"campaign_id":   campaignID.String(),
	})
}
