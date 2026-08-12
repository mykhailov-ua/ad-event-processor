package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/bidshard/ad-event-processor/internal/controlplane/adminapi"
	db "github.com/bidshard/ad-event-processor/internal/domain/db"
	"github.com/bidshard/ad-event-processor/internal/testutil"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

func TestPostbacksAdminAPIIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	cfgPostgres := testutil.DefaultPostgresConfig()
	cfgPostgres.MigrationDirs = []string{testutil.AdsMigrationsDir(), testutil.BillingMigrationsDir()}
	dbPool, cleanupDB := testutil.SetupPostgres(t, cfgPostgres)
	defer cleanupDB()

	customerID := uuid.New()
	_, err := dbPool.Exec(ctx, "INSERT INTO customers (id, name, balance, currency) VALUES ($1, $2, 0, 'USD')", customerID, "Customer")
	require.NoError(t, err)

	campaignID := uuid.New()
	_, err = dbPool.Exec(ctx, `
		INSERT INTO campaigns (id, name, status, customer_id)
		VALUES ($1, 'Campaign', 'ACTIVE', $2)`,
		campaignID,
		customerID,
	)
	require.NoError(t, err)

	key := []byte("postback-encryption-secret-key32")
	handler := &adminapi.PostbackHTTPHandlers{
		Pool:          dbPool,
		EncryptionKey: key,
	}

	mux := http.NewServeMux()
	handler.Register(mux)

	configReq := adminapi.UpdatePostbackConfigRequest{
		Provider:    "facebook",
		UrlTemplate: "https://mock.com",
		ApiToken:    "token123",
		TargetEvent: "conversion",
	}
	bodyBytes, err := json.Marshal(configReq)
	require.NoError(t, err)

	req := httptest.NewRequest("PUT", "/api/v1/postbacks/config/"+campaignID.String(), bytes.NewReader(bodyBytes))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	q := db.New(dbPool)
	configEntry, err := q.GetPostbackConfig(ctx, pgtype.UUID{Bytes: campaignID, Valid: true})
	require.NoError(t, err)
	require.Equal(t, "facebook", configEntry.Provider)
	require.Equal(t, "https://mock.com", configEntry.UrlTemplate)

	req = httptest.NewRequest("GET", "/api/v1/postbacks/config", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var configs []adminapi.PostbackConfigDTO
	err = json.NewDecoder(rec.Body).Decode(&configs)
	require.NoError(t, err)
	require.Len(t, configs, 1)
	require.Equal(t, campaignID.String(), configs[0].CampaignID)

	_, err = q.InsertPostbackDLQ(ctx, db.InsertPostbackDLQParams{
		OutboxEventID: 1001,
		CampaignID:    pgtype.UUID{Bytes: campaignID, Valid: true},
		ClickID:       "click_dlq_abc",
		EventType:     "conversion",
		Payload:       []byte(`{"click_id": "click_dlq_abc"}`),
		FailuresCount: 5,
		LastError:     pgtype.Text{String: "timeout error", Valid: true},
		Status:        "FAILED",
	})
	require.NoError(t, err)

	req = httptest.NewRequest("GET", "/api/v1/postbacks/dlq", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var dlqs []adminapi.PostbackDlqDTO
	err = json.NewDecoder(rec.Body).Decode(&dlqs)
	require.NoError(t, err)
	require.Len(t, dlqs, 1)
	require.Equal(t, "click_dlq_abc", dlqs[0].ClickID)
	require.Equal(t, "FAILED", dlqs[0].Status)

	dlqIDStr := strconv.FormatInt(dlqs[0].ID, 10)
	req = httptest.NewRequest("POST", "/api/v1/postbacks/dlq/"+dlqIDStr+"/retry", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	dlqUpdated, err := q.GetPostbackDLQ(ctx, dlqs[0].ID)
	require.NoError(t, err)
	require.Equal(t, "RETRIED", dlqUpdated.Status)

	outboxEvents, err := q.GetPendingPostbackEventsForUpdate(ctx, 10)
	require.NoError(t, err)
	require.Len(t, outboxEvents, 1)
	require.Equal(t, "SEND_POSTBACK", outboxEvents[0].EventType)
}
