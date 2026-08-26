package controlplane

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"ad-event-processor/internal/costsync"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/testutil"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

func TestCostSyncHandlers_microsoftAdsExtraConfigRoundTrip(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: cost sync microsoft_ads extra_config encrypt round-trip")
	}

	ctx := context.Background()
	cfg := testutil.DefaultPostgresConfig()
	cfg.MigrationDirs = []string{testutil.AdsMigrationsDir(), testutil.BillingMigrationsDir()}
	pool, cleanup := testutil.SetupPostgres(t, cfg)
	defer cleanup()

	customerID := uuid.New()
	_, err := pool.Exec(ctx, `INSERT INTO customers (id, name, balance, currency) VALUES ($1, 'cust', 0, 'USD')`, customerID)
	require.NoError(t, err)

	key := []byte("postback-encryption-secret-key32")
	handler := &CostSyncHTTPHandlers{
		Pool:          pool,
		EncryptionKey: key,
	}
	mux := http.NewServeMux()
	handler.Register(mux)

	upsertBody, err := json.Marshal(UpsertCostSyncCredentialRequest{
		CustomerID:  customerID.String(),
		AccountID:   "acct-1",
		AccessToken: "oauth-access",
		ExtraConfig: map[string]string{
			"customer_id":     "998877",
			"developer_token": "dev-secret-token",
		},
	})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/cost-sync/credentials/microsoft_ads", bytes.NewReader(upsertBody))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	var upsertResp CostSyncCredentialDTO
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &upsertResp))
	require.Equal(t, "998877", upsertResp.Extra["customer_id"])
	require.Empty(t, upsertResp.Extra["developer_token"])
	require.True(t, upsertResp.ExtraSet["developer_token"])

	req = httptest.NewRequest(http.MethodGet, "/api/v1/cost-sync/credentials?customer_id="+customerID.String(), http.NoBody)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var listed []CostSyncCredentialDTO
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &listed))
	require.Len(t, listed, 1)
	require.Equal(t, "998877", listed[0].Extra["customer_id"])
	require.True(t, listed[0].ExtraSet["developer_token"])

	storedRow, err := db.New(pool).GetCostSyncCredential(ctx, db.GetCostSyncCredentialParams{
		CustomerID: pgtype.UUID{Bytes: customerID, Valid: true},
		Network:    "microsoft_ads",
	})
	require.NoError(t, err)

	worker := costsync.NewWorker(pool, key)
	cred, err := worker.DecryptCredential(storedRow)
	require.NoError(t, err)
	require.Equal(t, "998877", cred.ExtraConfig["customer_id"])
	require.Equal(t, "dev-secret-token", cred.ExtraConfig["developer_token"])

	patchBody, err := json.Marshal(UpsertCostSyncCredentialRequest{
		CustomerID: customerID.String(),
		AccountID:  "acct-1",
		ExtraConfig: map[string]string{
			"customer_id":     "112233",
			"developer_token": "",
		},
	})
	require.NoError(t, err)
	req = httptest.NewRequest(http.MethodPut, "/api/v1/cost-sync/credentials/microsoft_ads", bytes.NewReader(patchBody))
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	storedRow, err = db.New(pool).GetCostSyncCredential(ctx, db.GetCostSyncCredentialParams{
		CustomerID: pgtype.UUID{Bytes: customerID, Valid: true},
		Network:    "microsoft_ads",
	})
	require.NoError(t, err)
	cred, err = worker.DecryptCredential(storedRow)
	require.NoError(t, err)
	require.Equal(t, "112233", cred.ExtraConfig["customer_id"])
	require.Equal(t, "dev-secret-token", cred.ExtraConfig["developer_token"])
}
