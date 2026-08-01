package costsync

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	db "espx/internal/domain/db"
	"espx/internal/testutil"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func setupCostSyncDB(t testing.TB) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()
	cfg := testutil.DefaultPostgresConfig()
	cfg.MigrationDirs = []string{testutil.AdsMigrationsDir()}
	pool, cleanup := testutil.SetupPostgres(t, cfg)
	t.Cleanup(cleanup)

	_, err := pool.Exec(ctx, `
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
	return pool
}

func seedCustomerCampaign(t testing.TB, pool *pgxpool.Pool) (uuid.UUID, uuid.UUID) {
	t.Helper()
	ctx := context.Background()
	customerID := uuid.New()
	campaignID := uuid.New()
	_, err := pool.Exec(ctx, `INSERT INTO customers (id, name, balance, currency) VALUES ($1, 'test', 0, 'USD')`, customerID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO campaigns (id, name, status, customer_id) VALUES ($1, 'c1', 'ACTIVE', $2)`, campaignID, customerID)
	require.NoError(t, err)
	return customerID, campaignID
}

func TestCurrency_EURToUSD(t *testing.T) {
	got := ConvertEURToUSD(100 * microUnit)
	require.Equal(t, int64(110_000_000), got)
}

func TestOAuthRefresh_Meta(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": "new-token",
			"expires_in":   3600,
		})
	}))
	defer srv.Close()

	token, expires, err := refreshMetaOAuth(context.Background(), &http.Client{
		Transport: roundTripRewriteHost(srv.URL, nil),
	}, "app", "secret", Credential{RefreshToken: "rt"})
	require.NoError(t, err)
	require.Equal(t, "new-token", token)
	require.True(t, expires.After(time.Now()))
}

func TestOAuthRefresh_GoogleHttptest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "application/x-www-form-urlencoded", r.Header.Get("Content-Type"))
		_ = r.ParseForm()
		require.Equal(t, "refresh_token", r.FormValue("grant_type"))
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": "google-new",
			"expires_in":   1800,
		})
	}))
	defer srv.Close()

	token, expires, err := refreshGoogleOAuth(context.Background(), &http.Client{
		Transport: roundTripRewriteHost(srv.URL, nil),
	}, "cid", "sec", Credential{RefreshToken: "rt"})
	require.NoError(t, err)
	require.Equal(t, "google-new", token)
	require.True(t, expires.After(time.Now()))
}

func roundTripRewriteHost(target string, base http.RoundTripper) http.RoundTripper {
	if base == nil {
		base = http.DefaultTransport
	}
	return roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		req = req.Clone(req.Context())
		req.URL.Scheme = "http"
		req.URL.Host = strings.TrimPrefix(target, "http://")
		return base.RoundTrip(req)
	})
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestIdempotency_DuplicateImport(t *testing.T) {
	pool := setupCostSyncDB(t)
	ctx := context.Background()
	customerID, campaignID := seedCustomerCampaign(t, pool)

	memCH := &MemorySnapshotInserter{}
	worker := NewWorker(pool, []byte("postback-encryption-secret-key32"), WithMemorySnapshots(memCH))

	date := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	lines := []CostLine{{
		CustomerID:  customerID,
		CampaignID:  campaignID,
		Date:        date,
		Network:     "facebook",
		PlacementID: "ad-1",
		LineType:    LineTypeSpend,
		AmountMicro: 5_000_000,
		Currency:    "USD",
	}}
	_, _, err := worker.persistLines(ctx, lines, date)
	require.NoError(t, err)
	_, _, err = worker.persistLines(ctx, lines, date)
	require.NoError(t, err)

	var count int
	err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM campaign_costs WHERE campaign_id = $1`, campaignID).Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 1, count)
}

func TestFault_DuplicateReportLedgerBalanced(t *testing.T) {
	if testing.Short() {
		t.Skip("fault integration test")
	}
	pool := setupCostSyncDB(t)
	ctx := context.Background()
	customerID, campaignID := seedCustomerCampaign(t, pool)
	date := time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC)

	_, err := pool.Exec(ctx, `
		INSERT INTO balance_ledger (customer_id, campaign_id, amount, type)
		VALUES ($1, $2, -3_000_000, 'FEE')`, customerID, campaignID)
	require.NoError(t, err)

	lines := []CostLine{{
		CustomerID:  customerID,
		CampaignID:  campaignID,
		Date:        date,
		Network:     "taboola",
		PlacementID: "site-a",
		LineType:    LineTypeSpend,
		AmountMicro: 5_000_000,
		Currency:    "USD",
	}}

	worker := NewWorker(pool, []byte("postback-encryption-secret-key32"))
	_, _, err = worker.persistLines(ctx, lines, date)
	require.NoError(t, err)
	_, _, err = worker.persistLines(ctx, lines, date)
	require.NoError(t, err)

	require.NoError(t, worker.reconcileCampaigns(ctx, lines, date))
	require.NoError(t, worker.reconcileCampaigns(ctx, lines, date))

	var ledgerCount int
	err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM balance_ledger WHERE type = 'RECONCILIATION_ADJUST' AND campaign_id = $1`, campaignID).Scan(&ledgerCount)
	require.NoError(t, err)
	require.Equal(t, 1, ledgerCount)

	var costCount int
	err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM campaign_costs WHERE campaign_id = $1`, campaignID).Scan(&costCount)
	require.NoError(t, err)
	require.Equal(t, 1, costCount)

	testutil.LogFaultProof(t, "cost_sync_duplicate_report", map[string]string{"ledger_balanced": "true"})
}

func TestRSOC_TonicGoldenFixture(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	require.True(t, ok)
	fixturePath := filepath.Join(filepath.Dir(filename), "testdata", "tonic_epc_daily.json")
	raw, err := os.ReadFile(fixturePath)
	require.NoError(t, err)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(raw)
	}))
	defer srv.Close()

	customerID := uuid.New()
	lines, err := fetchTonicRSOCCosts(context.Background(), srv.Client(), srv.URL, Credential{CustomerID: customerID}, time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	require.Len(t, lines, 1)
	require.Equal(t, LineTypeRevenue, lines[0].LineType)
	require.Equal(t, int64(12_500_000), lines[0].AmountMicro)
}

func TestRSOC_System1GoldenFixture(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	require.True(t, ok)
	fixturePath := filepath.Join(filepath.Dir(filename), "testdata", "system1_hourly.json")
	raw, err := os.ReadFile(fixturePath)
	require.NoError(t, err)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(raw)
	}))
	defer srv.Close()

	customerID := uuid.New()
	lines, err := fetchSystem1RSOCCosts(context.Background(), srv.Client(), srv.URL, Credential{CustomerID: customerID, APIKey: "k"}, time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	require.Len(t, lines, 1)
	require.Equal(t, int64(8_750_000), lines[0].AmountMicro)
}

func TestFacebookProvider_Httptest(t *testing.T) {
	customerID := uuid.New()
	campaignID := uuid.New()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]string{{
				"campaign_id": campaignID.String(),
				"adset_id":    "as1",
				"ad_id":       "ad1",
				"spend":       "12.50",
				"date_start":  "2026-07-01",
			}},
		})
	}))
	defer srv.Close()

	lines, err := fetchFacebookCosts(context.Background(), srv.Client(), srv.URL, Credential{
		CustomerID:  customerID,
		AccessToken: "tok",
		AccountID:   "act_123",
	}, time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	require.Len(t, lines, 1)
	require.Equal(t, int64(12_500_000), lines[0].AmountMicro)
}

func TestWorker_AdvisoryLock(t *testing.T) {
	pool := setupCostSyncDB(t)
	ctx := context.Background()

	conn1, err := pool.Acquire(ctx)
	require.NoError(t, err)
	defer conn1.Release()

	var ok1 bool
	err = conn1.QueryRow(ctx, `SELECT pg_try_advisory_lock($1)`, costSyncAdvisoryLockKey).Scan(&ok1)
	require.NoError(t, err)
	require.True(t, ok1)

	conn2, err := pool.Acquire(ctx)
	require.NoError(t, err)
	defer conn2.Release()

	var ok2 bool
	err = conn2.QueryRow(ctx, `SELECT pg_try_advisory_lock($1)`, costSyncAdvisoryLockKey).Scan(&ok2)
	require.NoError(t, err)
	require.False(t, ok2)

	_, err = conn1.Exec(ctx, `SELECT pg_advisory_unlock($1)`, costSyncAdvisoryLockKey)
	require.NoError(t, err)

	err = conn2.QueryRow(ctx, `SELECT pg_try_advisory_lock($1)`, costSyncAdvisoryLockKey).Scan(&ok2)
	require.NoError(t, err)
	require.True(t, ok2)
	_, _ = conn2.Exec(ctx, `SELECT pg_advisory_unlock($1)`, costSyncAdvisoryLockKey)
}

func TestUpsertCredential_EncryptionRoundTrip(t *testing.T) {
	pool := setupCostSyncDB(t)
	ctx := context.Background()
	customerID, _ := seedCustomerCampaign(t, pool)

	access, refresh, api, err := EncryptCredentialFields([]byte("postback-encryption-secret-key32"), "access", "refresh", "apikey")
	require.NoError(t, err)

	_, err = db.New(pool).UpsertCostSyncCredential(ctx, db.UpsertCostSyncCredentialParams{
		CustomerID:            pgtype.UUID{Bytes: customerID, Valid: true},
		Network:               "facebook",
		AccountID:             "act_1",
		AccessTokenEncrypted:  access,
		RefreshTokenEncrypted: refresh,
		ApiKeyEncrypted:       api,
		ExtraConfig:           []byte(`{}`),
	})
	require.NoError(t, err)

	row, err := db.New(pool).GetCostSyncCredential(ctx, db.GetCostSyncCredentialParams{
		CustomerID: pgtype.UUID{Bytes: customerID, Valid: true},
		Network:    "facebook",
	})
	require.NoError(t, err)

	worker := NewWorker(pool, []byte("postback-encryption-secret-key32"))
	cred, err := worker.decryptCredential(row)
	require.NoError(t, err)
	require.Equal(t, "access", cred.AccessToken)
	require.Equal(t, "refresh", cred.RefreshToken)
	require.Equal(t, "apikey", cred.APIKey)
}
