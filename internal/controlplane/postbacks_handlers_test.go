package controlplane

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/testutil"

	"ad-event-processor/internal/postback"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

func TestPostbackConfig_PreservesTokenOnEmptyUpdate(t *testing.T) {
	ctx := context.Background()
	cfgPostgres := testutil.DefaultPostgresConfig()
	cfgPostgres.MigrationDirs = []string{testutil.AdsMigrationsDir(), testutil.BillingMigrationsDir()}
	pool, cleanup := testutil.SetupPostgres(t, cfgPostgres)
	defer cleanup()

	customerID := uuid.New()
	campaignID := uuid.New()
	_, err := pool.Exec(ctx, "INSERT INTO customers (id, name, balance, currency) VALUES ($1, 'c', 0, 'USD')", customerID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO campaigns (id, name, status, customer_id) VALUES ($1, 'Camp', 'ACTIVE', $2)`, campaignID, customerID)
	require.NoError(t, err)

	key := []byte("postback-encryption-secret-key32")
	h := &PostbackHTTPHandlers{Pool: pool, EncryptionKey: key}
	mux := http.NewServeMux()
	h.Register(mux)

	put := func(body UpdatePostbackConfigRequest) *httptest.ResponseRecorder {
		raw, err := json.Marshal(body)
		require.NoError(t, err)
		req := httptest.NewRequest(http.MethodPut, "/api/v1/postbacks/config/"+campaignID.String(), bytes.NewReader(raw))
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		return rec
	}

	rec := put(UpdatePostbackConfigRequest{
		Provider:    "facebook",
		URLTemplate: "999888777",
		APIToken:    "secret-token",
		TargetEvent: "conversion",
	})
	require.Equal(t, http.StatusOK, rec.Code)

	rec = put(UpdatePostbackConfigRequest{
		Provider:    "facebook",
		URLTemplate: "999888777",
		TargetEvent: "conversion",
	})
	require.Equal(t, http.StatusOK, rec.Code)

	q := db.New(pool)
	cfg, err := q.GetPostbackConfig(ctx, pgtype.UUID{Bytes: campaignID, Valid: true})
	require.NoError(t, err)
	require.NotEmpty(t, cfg.ApiTokenEncrypted)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/postbacks/config", http.NoBody)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var dtos []PostbackConfigDTO
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&dtos))
	require.Len(t, dtos, 1)
	require.True(t, dtos[0].HasAPIToken)
}

func TestPostbackConfig_CAPIRequiresToken(t *testing.T) {
	ctx := context.Background()
	cfgPostgres := testutil.DefaultPostgresConfig()
	cfgPostgres.MigrationDirs = []string{testutil.AdsMigrationsDir(), testutil.BillingMigrationsDir()}
	pool, cleanup := testutil.SetupPostgres(t, cfgPostgres)
	defer cleanup()

	customerID := uuid.New()
	campaignID := uuid.New()
	_, err := pool.Exec(ctx, "INSERT INTO customers (id, name, balance, currency) VALUES ($1, 'c', 0, 'USD')", customerID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO campaigns (id, name, status, customer_id) VALUES ($1, 'Camp', 'ACTIVE', $2)`, campaignID, customerID)
	require.NoError(t, err)

	h := &PostbackHTTPHandlers{Pool: pool, EncryptionKey: []byte("postback-encryption-secret-key32")}
	mux := http.NewServeMux()
	h.Register(mux)

	body, err := json.Marshal(UpdatePostbackConfigRequest{
		Provider:    "google",
		URLTemplate: "customers/1/conversionActions/2",
		TargetEvent: "conversion",
	})
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/postbacks/config/"+campaignID.String(), bytes.NewReader(body))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestPostbackConfig_DryRunWebhook(t *testing.T) {
	ctx := context.Background()
	cfgPostgres := testutil.DefaultPostgresConfig()
	cfgPostgres.MigrationDirs = []string{testutil.AdsMigrationsDir(), testutil.BillingMigrationsDir()}
	pool, cleanup := testutil.SetupPostgres(t, cfgPostgres)
	defer cleanup()

	customerID := uuid.New()
	campaignID := uuid.New()
	_, err := pool.Exec(ctx, "INSERT INTO customers (id, name, balance, currency) VALUES ($1, 'c', 0, 'USD')", customerID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO campaigns (id, name, status, customer_id) VALUES ($1, 'Camp', 'ACTIVE', $2)`, campaignID, customerID)
	require.NoError(t, err)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	h := &PostbackHTTPHandlers{Pool: pool, EncryptionKey: []byte("postback-encryption-secret-key32")}
	mux := http.NewServeMux()
	h.Register(mux)

	putBody, err := json.Marshal(UpdatePostbackConfigRequest{
		Provider:    "webhook",
		URLTemplate: srv.URL + "?click_id={click_id}&payout={payout}",
		TargetEvent: "conversion",
	})
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/postbacks/config/"+campaignID.String(), bytes.NewReader(putBody))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, "put: %s", rec.Body.String())

	req = httptest.NewRequest(http.MethodPost, "/api/v1/postbacks/config/"+campaignID.String()+"/test", http.NoBody)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, "test: %s", rec.Body.String())

	var result postback.DryRunResult
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&result))
	require.True(t, result.OK, result.Error)
	require.Equal(t, "webhook", result.Provider)
	require.Contains(t, result.RenderedURL, "dry-run-click-id")
}
