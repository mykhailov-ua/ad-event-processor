package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bidshard/ad-event-processor/internal/controlplane/adminapi"
	"github.com/bidshard/ad-event-processor/internal/testutil"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegration_ReportsDashboardsViews_NoTierGate(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping reports dashboards views integration test")
	}

	ctx := context.Background()

	cfgPostgres := testutil.DefaultPostgresConfig()
	cfgPostgres.MigrationDirs = []string{testutil.AdsMigrationsDir(), testutil.BillingMigrationsDir()}
	dbPool, cleanupDB := testutil.SetupPostgres(t, cfgPostgres)
	defer cleanupDB()

	customerID := uuid.New()
	_, err := dbPool.Exec(ctx, "INSERT INTO customers (id, name, balance, currency) VALUES ($1, $2, 0, 'USD')", customerID, "Basic Customer")
	require.NoError(t, err)

	reportsHandler := &adminapi.ReportsHTTPHandlers{
		Pool: dbPool,
		ResolveForecastCustomerID: func(r *http.Request, bodyID *uuid.UUID) (*uuid.UUID, error) {
			if custIDStr := r.URL.Query().Get("customer_id"); custIDStr != "" {
				parsed, err := uuid.Parse(custIDStr)
				if err == nil {
					return &parsed, nil
				}
			}
			return nil, nil
		},
	}

	viewsHandler := &adminapi.ViewsHTTPHandlers{
		Service: adminapi.NewService(),
	}

	mux := http.NewServeMux()
	reportsHandler.Register(mux)
	viewsHandler.Register(mux)

	t.Run("Reports_Placements_AllowedWithoutSubscription", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/reports/placements?customer_id="+customerID.String(), nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		assert.NotEqual(t, http.StatusForbidden, w.Code)
	})

	t.Run("Reports_Keywords_AllowedWithoutSubscription", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/reports/keywords?customer_id="+customerID.String(), nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		assert.NotEqual(t, http.StatusForbidden, w.Code)
	})

	t.Run("Views_CRUD_AllowedWithoutSubscription", func(t *testing.T) {
		createReq := adminapi.CreateViewRequest{
			CustomerID: customerID.String(),
			Name:       "Saved View",
			ReportKey:  "placements",
			Spec:       json.RawMessage(`{}`),
		}
		body, _ := json.Marshal(createReq)
		req := httptest.NewRequest("POST", "/api/v1/views", bytes.NewReader(body))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)

		req = httptest.NewRequest("GET", "/api/v1/views?customer_id="+customerID.String(), nil)
		w = httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}
