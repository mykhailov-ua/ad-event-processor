package controlplane_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	ctrlhttp "ad-event-processor/internal/control/http"

	"ad-event-processor/internal/controlplane/authz"
	"ad-event-processor/internal/platformadmin"
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

	h := &platformadmin.TeamHTTPHandlers{
		Team: &platformadmin.TeamOverviewService{Pool: pool},
		SnapshotFromRequest: func(_ *http.Request) (authz.Snapshot, bool) {
			return authz.Snapshot{
				Permissions: map[string]struct{}{
					ctrlhttp.PermBillingRead:   {},
					ctrlhttp.PermCampaignsRead: {},
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

	var body platformadmin.TeamOverviewDTO
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, customerID.String(), body.CustomerID)
	require.Equal(t, int64(5000000), body.BalanceMicro)
	require.NotNil(t, body.License)
	require.Equal(t, "ACTIVE", body.License.State)
	require.Empty(t, body.Members)
	_ = time.Now()
}

func TestTeamHandlers_listMembersPagination(t *testing.T) {
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
	_, err := pool.Exec(ctx, `INSERT INTO customers (id, name, balance, currency) VALUES ($1,'TeamCo',5000000,'USD')`, customerID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `
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

	emails := []string{"alpha@teamco.test", "beta@teamco.test", "gamma@teamco.test"}
	for i, email := range emails {
		_, err = pool.Exec(ctx, `
			INSERT INTO users (id, email, password_hash, role, customer_id)
			VALUES ($1, $2, 'hash', 'MB', $3)`, uuid.New(), email, customerID)
		require.NoError(t, err, "insert member %d", i)
	}

	govStub := &teamGovStub{}
	h := &platformadmin.TeamHTTPHandlers{
		Team:       &platformadmin.TeamOverviewService{Pool: pool},
		Governance: govStub,
		ResolveCustomerID: func(_ *http.Request, _ *uuid.UUID) (uuid.UUID, error) {
			return customerID, nil
		},
	}
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/team/members?limit=2&offset=0", http.NoBody)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	var page platformadmin.TeamMembersListResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &page))
	require.Equal(t, int64(3), page.Total)
	require.Equal(t, 2, page.Limit)
	require.Equal(t, 0, page.Offset)
	require.Len(t, page.Items, 2)
	require.Equal(t, "alpha@teamco.test", page.Items[0].Email)
	require.Equal(t, "beta@teamco.test", page.Items[1].Email)
	require.NotEmpty(t, page.Items[0].CreatedAtDisplay)

	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/team/members?limit=2&offset=2", http.NoBody)
	rec2 := httptest.NewRecorder()
	mux.ServeHTTP(rec2, req2)
	require.Equal(t, http.StatusOK, rec2.Code, rec2.Body.String())

	var page2 platformadmin.TeamMembersListResponse
	require.NoError(t, json.Unmarshal(rec2.Body.Bytes(), &page2))
	require.Len(t, page2.Items, 1)
	require.Equal(t, "gamma@teamco.test", page2.Items[0].Email)
}
