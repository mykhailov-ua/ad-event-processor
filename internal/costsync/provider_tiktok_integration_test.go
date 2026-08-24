package costsync

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/testutil"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

func TestTikTokCostSync_integration(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: tiktok cost sync oauth refresh + report fixture (run make test-integration)")
	}

	_, filename, _, ok := runtime.Caller(0)
	require.True(t, ok)
	fixtureDir := filepath.Join(filepath.Dir(filename), "testdata")
	reportFixture, err := os.ReadFile(filepath.Join(fixtureDir, "tiktok_report_integrated.json"))
	require.NoError(t, err)
	refreshFixture, err := os.ReadFile(filepath.Join(fixtureDir, "tiktok_oauth_refresh.json"))
	require.NoError(t, err)

	ctx := context.Background()
	cfg := testutil.DefaultPostgresConfig()
	cfg.MigrationDirs = []string{testutil.AdsMigrationsDir()}
	pool, cleanup := testutil.SetupPostgres(t, cfg)
	defer cleanup()

	_, err = pool.Exec(ctx, `
		CREATE SCHEMA IF NOT EXISTS billing;
		CREATE TABLE IF NOT EXISTS billing.customer_subscriptions (
			customer_id UUID PRIMARY KEY,
			plan_code TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'active',
			period_start TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			period_end TIMESTAMPTZ,
			overrides_json JSONB NOT NULL DEFAULT '{}',
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
	`)
	require.NoError(t, err)
	customerID := uuid.New()
	campKey := "camp-99"
	tiktokCampaignID := uuid.NewSHA1(customerID, []byte("tiktok:"+campKey))

	_, err = pool.Exec(ctx, `INSERT INTO customers (id, name, balance, currency) VALUES ($1, 'test', 0, 'USD')`, customerID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO campaigns (id, name, status, customer_id) VALUES ($1, 'tiktok-camp', 'ACTIVE', $2)`, tiktokCampaignID, customerID)
	require.NoError(t, err)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/open_api/v1.3/oauth2/refresh_token/":
			require.Equal(t, http.MethodPost, r.Method)
			require.Equal(t, "application/json", r.Header.Get("Content-Type"))
			var body map[string]string
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			require.Equal(t, "tt-app", body["app_id"])
			require.Equal(t, "tt-secret", body["secret"])
			require.Equal(t, "refresh_token", body["grant_type"])
			require.Equal(t, "refresh-old", body["refresh_token"])
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(refreshFixture)
		case "/open_api/v1.3/report/integrated/get/":
			require.Equal(t, "refreshed-tok", r.Header.Get("Access-Token"))
			require.Equal(t, "adv-42", r.URL.Query().Get("advertiser_id"))
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(reportFixture)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	accessEnc, refreshEnc, _, err := EncryptCredentialFields(
		[]byte("postback-encryption-secret-key32"),
		"expired-tok",
		"refresh-old",
		"",
	)
	require.NoError(t, err)

	credRow, err := db.New(pool).UpsertCostSyncCredential(ctx, db.UpsertCostSyncCredentialParams{
		CustomerID:            pgtype.UUID{Bytes: customerID, Valid: true},
		Network:               "tiktok",
		AccountID:             "adv-42",
		AccessTokenEncrypted:  accessEnc,
		RefreshTokenEncrypted: refreshEnc,
		ExtraConfig:           []byte(`{}`),
		TokenExpiresAt:        pgtype.Timestamptz{Time: time.Now().Add(-time.Hour), Valid: true},
	})
	require.NoError(t, err)

	worker := NewWorker(
		pool,
		[]byte("postback-encryption-secret-key32"),
		WithOAuth(OAuthConfig{TikTokAppID: "tt-app", TikTokAppSecret: "tt-secret"}),
		WithNetworkBaseURL("tiktok", srv.URL+"/open_api/v1.3"),
		WithMemorySnapshots(&MemorySnapshotInserter{}),
	)
	worker.httpClient = &http.Client{Transport: roundTripRewriteHost(srv.URL, nil)}

	syncDate := time.Date(2026, 8, 24, 0, 0, 0, 0, time.UTC)
	require.NoError(t, worker.syncCredential(ctx, credRow, syncDate, "integration"))

	var costCount int
	err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM campaign_costs WHERE campaign_id = $1`, tiktokCampaignID).Scan(&costCount)
	require.NoError(t, err)
	require.Equal(t, 1, costCount)

	var amountMicro int64
	err = pool.QueryRow(ctx, `SELECT amount_micro FROM campaign_costs WHERE campaign_id = $1`, tiktokCampaignID).Scan(&amountMicro)
	require.NoError(t, err)
	require.Equal(t, int64(12_500_000), amountMicro)

	updated, err := db.New(pool).GetCostSyncCredential(ctx, db.GetCostSyncCredentialParams{
		CustomerID: pgtype.UUID{Bytes: customerID, Valid: true},
		Network:    "tiktok",
	})
	require.NoError(t, err)
	decrypted, err := worker.DecryptCredential(updated)
	require.NoError(t, err)
	require.Equal(t, "refreshed-tok", decrypted.AccessToken)
	require.Equal(t, "refresh-rotated", decrypted.RefreshToken)
}
