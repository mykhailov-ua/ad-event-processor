package controlplane_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"ad-event-processor/internal/controlplane"
	"ad-event-processor/internal/controlplane/authz"
	"ad-event-processor/internal/testutil"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestTeamHandlers_overview(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}

	ctx := context.Background()
	cfg := testutil.DefaultPostgresConfig()
	cfg.MigrationDirs = []string{
		testutil.AdsMigrationsDir(),
		testutil.BillingMigrationsDir(),
	}
	pool, cleanup := testutil.SetupPostgres(t, cfg)
	defer cleanup()

	customerID := uuid.New()
	userID := uuid.New()
	_, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS users (
			id UUID PRIMARY KEY,
			email TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			role TEXT NOT NULL,
			customer_id UUID,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			is_blocked BOOLEAN NOT NULL DEFAULT FALSE,
			email_verified BOOLEAN NOT NULL DEFAULT TRUE
		)`)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO customers (id, name, balance, currency) VALUES ($1,'TeamCo',5000000,'USD')`, customerID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `
		INSERT INTO users (id, email, password_hash, role, customer_id)
		VALUES ($1, 'lead@teamco.test', 'hash', 'TL', $2)`, userID, customerID)
	require.NoError(t, err)

	deploymentID := uuid.New()
	licenseID := uuid.New()
	_, err = pool.Exec(ctx, `
		INSERT INTO billing.license_status (deployment_id, license_id, plan_code, valid_until, state, entitlements_json, last_verified_at)
		VALUES ($1, $2, 'enterprise', NOW() + interval '30 days', 'ACTIVE', '{}', NOW())`,
		deploymentID, licenseID)
	require.NoError(t, err)

	h := &controlplane.TeamHTTPHandlers{
		Team: &controlplane.TeamOverviewService{Pool: pool},
		SnapshotFromRequest: func(_ *http.Request) (authz.Snapshot, bool) {
			return authz.Snapshot{
				Permissions: map[string]struct{}{
					authz.PermBillingRead:   {},
					authz.PermCampaignsRead: {},
				},
			}, true
		},
		ResolveCustomerID: func(_ *http.Request, _ *uuid.UUID) (uuid.UUID, error) {
			return customerID, nil
		},
	}
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/team/overview", http.NoBody)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	var body controlplane.TeamOverviewDTO
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, customerID.String(), body.CustomerID)
	require.Equal(t, int64(5000000), body.BalanceMicro)
	require.NotNil(t, body.License)
	require.Equal(t, "ACTIVE", body.License.State)
	require.Len(t, body.Members, 1)
	require.Equal(t, userID.String(), body.Members[0].UserID)
	_ = time.Now()
}
