package controlplane

import (
	"ad-event-processor/internal/opsadmin"
	"context"
	"encoding/json"
	"strconv"
	"testing"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/database"
	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupCPAHeldOutDB(t *testing.T) (*Service, func()) {
	t.Helper()
	if testing.Short() {
		t.Skip("CPA held-out integration: run without -short (Docker testcontainers)")
	}
	pool, cleanupDB := database.SetupTestDB(t)
	svc := &Service{pool: pool, cfg: &config.Config{}}
	return svc, cleanupDB
}

func seedCustomerCampaign(t *testing.T, ctx context.Context, pool *pgxpool.Pool, name string) (uuid.UUID, uuid.UUID) {
	t.Helper()
	customerID := uuid.New()
	campaignID := uuid.New()
	_, err := pool.Exec(ctx, `
		INSERT INTO customers (id, name, balance, currency) VALUES ($1, $2, 0, 'USD')`,
		domain.ToUUID(customerID), name+"-cust")
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `
		INSERT INTO campaigns (id, name, budget_limit, current_spend, status, customer_id, pacing_mode, timezone, freq_window)
		VALUES ($1, $2, 1000000, 0, 'ACTIVE', $3, 'ASAP', 'UTC', 86400)`,
		domain.ToUUID(campaignID), name, domain.ToUUID(customerID))
	require.NoError(t, err)
	return customerID, campaignID
}

func TestCPA_HeldOut_ListDLQInbox_capiVsPostbackFilter(t *testing.T) {
	svc, cleanup := setupCPAHeldOutDB(t)
	defer cleanup()
	ctx := context.Background()
	reader := newOpsReader(svc)

	_, campWebhook := seedCustomerCampaign(t, ctx, svc.pool, "heldout-webhook")
	_, campCAPI := seedCustomerCampaign(t, ctx, svc.pool, "heldout-capi")
	q := db.New(svc.pool)

	payload, err := json.Marshal(map[string]string{"click_id": "clk-1"})
	require.NoError(t, err)

	require.NoError(t, q.UpsertPostbackConfig(ctx, db.UpsertPostbackConfigParams{
		CampaignID:  pgtype.UUID{Bytes: campWebhook, Valid: true},
		Provider:    "webhook",
		UrlTemplate: "https://example.com/pb",
		TargetEvent: "conversion",
	}))
	require.NoError(t, q.UpsertPostbackConfig(ctx, db.UpsertPostbackConfigParams{
		CampaignID:  pgtype.UUID{Bytes: campCAPI, Valid: true},
		Provider:    "facebook",
		UrlTemplate: "https://graph.facebook.com/events",
		TargetEvent: "conversion",
	}))

	webhookDLQ, err := q.InsertPostbackDLQ(ctx, db.InsertPostbackDLQParams{
		OutboxEventID: 8001,
		CampaignID:    pgtype.UUID{Bytes: campWebhook, Valid: true},
		ClickID:       "clk-webhook",
		EventType:     "conversion",
		Payload:       payload,
		FailuresCount: 1,
		LastError:     pgtype.Text{String: "timeout", Valid: true},
		Status:        "FAILED",
	})
	require.NoError(t, err)
	capiDLQ, err := q.InsertPostbackDLQ(ctx, db.InsertPostbackDLQParams{
		OutboxEventID: 8002,
		CampaignID:    pgtype.UUID{Bytes: campCAPI, Valid: true},
		ClickID:       "clk-capi",
		EventType:     "conversion",
		Payload:       payload,
		FailuresCount: 2,
		LastError:     pgtype.Text{String: "401", Valid: true},
		Status:        "FAILED",
	})
	require.NoError(t, err)

	postbackOnly, err := reader.ListDLQInbox(ctx, "postback", "", 50)
	require.NoError(t, err)
	require.Len(t, postbackOnly.Items, 1)
	assert.Equal(t, "postback", postbackOnly.Items[0].Source)
	assert.Equal(t, webhookDLQ.ID, mustParseInt64(t, postbackOnly.Items[0].ID))

	capiOnly, err := reader.ListDLQInbox(ctx, "capi", "", 50)
	require.NoError(t, err)
	require.Len(t, capiOnly.Items, 1)
	assert.Equal(t, "capi", capiOnly.Items[0].Source)
	assert.Equal(t, capiDLQ.ID, mustParseInt64(t, capiOnly.Items[0].ID))
}

func TestCPA_HeldOut_ListConsentProofs_filtersByUser(t *testing.T) {
	svc, cleanup := setupCPAHeldOutDB(t)
	defer cleanup()
	ctx := context.Background()
	reader := newOpsReader(svc)

	require.NoError(t, svc.RecordConsent(ctx, opsadmin.ConsentRecord{
		UserID:   "heldout-user-a",
		Purposes: domain.ConsentPurposeAdStorage,
		Source:   "cmp",
	}))
	require.NoError(t, svc.RecordConsent(ctx, opsadmin.ConsentRecord{
		UserID:   "heldout-user-b",
		Purposes: domain.ConsentPurposeAnalytics,
		Source:   "web",
	}))

	filtered, err := reader.ListConsentProofs(ctx, "heldout-user-a", "", 50)
	require.NoError(t, err)
	require.Len(t, filtered.Items, 1)
	assert.Equal(t, "cmp", filtered.Items[0].Source)
	assert.True(t, filtered.Items[0].AdStorage)

	all, err := reader.ListConsentProofs(ctx, "", "", 50)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(all.Items), 2)
}

func TestCPA_HeldOut_ListDomainRotation_dmrCounts(t *testing.T) {
	svc, cleanup := setupCPAHeldOutDB(t)
	defer cleanup()
	ctx := context.Background()
	reader := newOpsReader(svc)

	poolID := uuid.MustParse("55555555-5555-4555-8555-555555555555")
	host := "trk-m8.example"
	_, err := svc.pool.Exec(ctx, `
		INSERT INTO domain_pools (id, name) VALUES ($1, 'heldout-rotation-pool')
		ON CONFLICT (name) DO NOTHING`, poolID)
	require.NoError(t, err)
	_, err = svc.pool.Exec(ctx, `
		INSERT INTO domain_health_status (hostname, role, health_status, ssl_status)
		VALUES ($1, 'tracking', 'healthy', 'valid')
		ON CONFLICT (hostname) DO NOTHING`, host)
	require.NoError(t, err)
	_, err = svc.pool.Exec(ctx, `
		INSERT INTO domain_pool_domains (pool_id, hostname, sort_order, status)
		VALUES ($1, $2, 0, 'active')
		ON CONFLICT (hostname) DO UPDATE SET pool_id = EXCLUDED.pool_id`, poolID, host)
	require.NoError(t, err)

	_, campDMR := seedCustomerCampaign(t, ctx, svc.pool, "heldout-dmr")
	_, campPlain := seedCustomerCampaign(t, ctx, svc.pool, "heldout-plain")
	_, err = svc.pool.Exec(ctx, `UPDATE campaigns SET domain_pool_id = $1, dmr_enabled = true WHERE id = $2`,
		poolID, domain.ToUUID(campDMR))
	require.NoError(t, err)
	_, err = svc.pool.Exec(ctx, `UPDATE campaigns SET domain_pool_id = $1, dmr_enabled = false WHERE id = $2`,
		poolID, domain.ToUUID(campPlain))
	require.NoError(t, err)

	result, err := reader.ListDomainRotation(ctx)
	require.NoError(t, err)
	var dmrCount, activeCount int64
	for _, h := range result.Hosts {
		if h.Hostname == host {
			dmrCount = h.DmrCampaignCount
			activeCount = h.ActiveCampaignCount
			break
		}
	}
	assert.Equal(t, int64(1), dmrCount)
	assert.Equal(t, int64(2), activeCount)
}

func TestCPA_HeldOut_GetIncidentSnapshot_staleListsCampaigns(t *testing.T) {
	svc, cleanup := setupCPAHeldOutDB(t)
	defer cleanup()
	ctx := context.Background()
	reader := newOpsReader(svc)

	_, campID := seedCustomerCampaign(t, ctx, svc.pool, "heldout-incident")
	_, err := svc.pool.Exec(ctx, `UPDATE campaigns SET name = 'Held-out incident camp' WHERE id = $1`, domain.ToUUID(campID))
	require.NoError(t, err)

	snap, err := reader.GetIncidentSnapshot(ctx)
	require.NoError(t, err)
	assert.True(t, snap.StaleDashboard)
	require.NotEmpty(t, snap.AffectedCampaigns)
	found := false
	for _, c := range snap.AffectedCampaigns {
		if c.CampaignID == campID.String() {
			found = true
			assert.Equal(t, "Held-out incident camp", c.Name)
			break
		}
	}
	assert.True(t, found)
}

func TestCPA_HeldOut_RetryDLQInbox_postbackRequeuesOutbox(t *testing.T) {
	svc, cleanup := setupCPAHeldOutDB(t)
	defer cleanup()
	ctx := context.Background()
	reader := newOpsReader(svc)

	_, campaignID := seedCustomerCampaign(t, ctx, svc.pool, "heldout-retry")
	q := db.New(svc.pool)
	payload, err := json.Marshal(map[string]string{"click_id": "clk-retry"})
	require.NoError(t, err)

	dlqRow, err := q.InsertPostbackDLQ(ctx, db.InsertPostbackDLQParams{
		OutboxEventID: 9009,
		CampaignID:    pgtype.UUID{Bytes: campaignID, Valid: true},
		ClickID:       "clk-retry",
		EventType:     "conversion",
		Payload:       payload,
		FailuresCount: 3,
		LastError:     pgtype.Text{String: "timeout", Valid: true},
		Status:        "FAILED",
	})
	require.NoError(t, err)

	require.NoError(t, reader.RetryDLQInbox(ctx, "postback", strconv.FormatInt(dlqRow.ID, 10), "idem-heldout-retry"))

	var outboxCount int
	require.NoError(t, svc.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM outbox_events WHERE event_type = 'SEND_POSTBACK'`).Scan(&outboxCount))
	assert.Equal(t, 1, outboxCount)

	var status string
	require.NoError(t, svc.pool.QueryRow(ctx, `SELECT status FROM postback_dlq WHERE id = $1`, dlqRow.ID).Scan(&status))
	assert.Equal(t, "RETRIED", status)
}

func mustParseInt64(t *testing.T, s string) int64 {
	t.Helper()
	n, err := strconv.ParseInt(s, 10, 64)
	require.NoError(t, err)
	return n
}
