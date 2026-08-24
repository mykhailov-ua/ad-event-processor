package postback

import (
	"context"
	"encoding/json"
	"testing"

	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/testutil"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

func TestConversionPostbackEnqueuer_integration(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}
	ctx := context.Background()
	cfg := testutil.DefaultPostgresConfig()
	cfg.MigrationDirs = []string{testutil.AdsMigrationsDir(), testutil.BillingMigrationsDir()}
	pool, cleanup := testutil.SetupPostgres(t, cfg)
	defer cleanup()

	customerID := uuid.New()
	campaignID := uuid.New()
	_, err := pool.Exec(ctx, `INSERT INTO customers (id, name, balance, currency) VALUES ($1, 'c', 0, 'USD')`, customerID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO campaigns (id, name, status, customer_id) VALUES ($1, 'Camp', 'ACTIVE', $2)`, campaignID, customerID)
	require.NoError(t, err)

	q := db.New(pool)
	require.NoError(t, q.UpsertPostbackConfig(ctx, db.UpsertPostbackConfigParams{
		CampaignID:        pgtype.UUID{Bytes: campaignID, Valid: true},
		Provider:          "webhook",
		UrlTemplate:       "https://example.com/pb",
		ApiTokenEncrypted: []byte("x"),
		TargetEvent:       "conversion",
	}))

	enq := NewConversionPostbackEnqueuer(q)
	enq.OnBatchStored(ctx, []*domain.Event{
		{
			ClickID:    "click-enq-1",
			CampaignID: campaignID,
			Type:       "conversion",
			Payload:    []byte(`{"gclid":"G1","sub1":"fb"}`),
		},
		{
			ClickID:    "click-skip",
			CampaignID: campaignID,
			Type:       "click",
		},
	})

	events, err := q.GetPendingPostbackEventsForUpdate(ctx, db.GetPendingPostbackEventsForUpdateParams{
		Limit:   10,
		Column2: 120,
	})
	require.NoError(t, err)
	require.Len(t, events, 1)
	require.Equal(t, outboxEventSendPostback, events[0].EventType)

	var payload PostbackPayload
	require.NoError(t, json.Unmarshal(events[0].Payload, &payload))
	require.Equal(t, "click-enq-1", payload.ClickID)
	require.Equal(t, "G1", payload.GCLID)
	require.Equal(t, "fb", payload.SubID1)
}
