package postback

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/bidshard/ad-event-processor/internal/database"
	"github.com/bidshard/ad-event-processor/internal/domain"
	db "github.com/bidshard/ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

const n1PostbackBatchSize = 20

func legacyProcessBatchOutboxUpdates(ctx context.Context, w *PostbackWorker, claimed []claimedPostbackEvent) {
	for _, c := range claimed {
		if c.skip {
			_, _ = w.pool.Exec(ctx, "UPDATE outbox_events SET status = 'PROCESSED', processing_started_at = NULL WHERE id = $1", c.event.ID)
			continue
		}
		_, _ = w.pool.Exec(ctx, "UPDATE outbox_events SET status = 'PROCESSED', processing_started_at = NULL WHERE id = $1", c.event.ID)
	}
}

func seedPostbackBatchEvents(t testing.TB, pool *pgxpool.Pool, n int) ([]db.OutboxEvent, uuid.UUID) {
	t.Helper()
	ctx := context.Background()
	customerID := uuid.New()
	campaignID := uuid.New()
	_, err := pool.Exec(ctx, `
		INSERT INTO customers (id, name, balance, currency) VALUES ($1, 'pb-bench', 0, 'USD')`,
		domain.ToUUID(customerID))
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `
		INSERT INTO campaigns (id, name, budget_limit, current_spend, status, customer_id, pacing_mode, timezone, freq_window)
		VALUES ($1, 'pb-bench', 100000000, 0, 'ACTIVE', $2, 'ASAP', 'UTC', 86400)`,
		domain.ToUUID(campaignID), domain.ToUUID(customerID))
	require.NoError(t, err)

	q := db.New(pool)
	require.NoError(t, q.UpsertPostbackConfig(ctx, db.UpsertPostbackConfigParams{
		CampaignID:        domain.ToUUID(campaignID),
		Provider:          "webhook",
		UrlTemplate:       "https://example.com/pb",
		ApiTokenEncrypted: []byte{},
		TargetEvent:       "conversion",
	}))
	events := make([]db.OutboxEvent, 0, n)
	for i := range n {
		payload, err := json.Marshal(PostbackPayload{
			CustomerID: customerID,
			CampaignID: campaignID,
			ClickID:    fmt.Sprintf("click-%d", i),
			EventType:  "conversion",
		})
		require.NoError(t, err)
		ev, err := q.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
			EventType: "SEND_POSTBACK",
			Payload:   payload,
		})
		require.NoError(t, err)
		events = append(events, ev)
	}
	return events, campaignID
}

func TestN1Fix_PostbackConfigPreload_QueryCount(t *testing.T) {
	if testing.Short() {
		t.Skip("integration")
	}
	pool, counter, cleanup := database.SetupTestDBWithQueryCounter(t)
	defer cleanup()
	ctx := context.Background()

	events, campaignID := seedPostbackBatchEvents(t, pool, n1PostbackBatchSize)
	worker := NewPostbackWorker(pool, []byte("postback-encryption-secret-key32"))

	pgIDs := make([]pgtype.UUID, 1)
	pgIDs[0] = domain.ToUUID(campaignID)

	counter.Reset()
	for range events {
		_, err := db.New(pool).GetPostbackConfig(ctx, pgIDs[0])
		require.NoError(t, err)
	}
	before := counter.Snapshot()

	counter.Reset()
	_, err := worker.loadPostbackConfigs(ctx, []uuid.UUID{campaignID})
	require.NoError(t, err)
	after := counter.Snapshot()

	t.Logf("5_postback_config queries: before=%d after=%d (events=%d)", before, after, n1PostbackBatchSize)
	require.Equal(t, int64(n1PostbackBatchSize), before)
	require.Equal(t, int64(1), after)
}

func TestN1Fix_PostbackOutboxBatchUpdate_QueryCount(t *testing.T) {
	if testing.Short() {
		t.Skip("integration")
	}
	pool, counter, cleanup := database.SetupTestDBWithQueryCounter(t)
	defer cleanup()
	ctx := context.Background()

	events, _ := seedPostbackBatchEvents(t, pool, n1PostbackBatchSize)
	worker := NewPostbackWorker(pool, []byte("postback-encryption-secret-key32"))

	claimed := make([]claimedPostbackEvent, len(events))
	for i, ev := range events {
		claimed[i] = claimedPostbackEvent{event: ev}
	}

	counter.Reset()
	legacyProcessBatchOutboxUpdates(ctx, worker, claimed)
	before := counter.Snapshot()

	ids := make([]int64, len(events))
	for i, ev := range events {
		ids[i] = ev.ID
	}
	counter.Reset()
	worker.batchUpdateOutboxStatus(ctx, ids, nil, nil)
	after := counter.Snapshot()

	t.Logf("5_postback_outbox queries: before=%d after=%d (events=%d)", before, after, n1PostbackBatchSize)
	require.Equal(t, int64(n1PostbackBatchSize), before)
	require.Equal(t, int64(1), after)
}
