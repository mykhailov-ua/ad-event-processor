package controlplane

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/database"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSQLiIntegration_MaliciousInputs(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}

	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()
	redisClient, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	cfg := &config.Config{
		AdminAPIKey:       "test-secret",
		TokenSymmetricKey: "01234567890123456789012345678901",
	}
	authMW, tokenMaker := integrationTestAuth(t, redisClient, cfg)
	svc := newBareService(t, pool, []redis.UniversalClient{redisClient}, cfg)
	h := NewHandler(svc, cfg, authMW, nil, nil, nil)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	ctx := context.Background()
	safeID := uuid.New()
	require.NoError(t, svc.CreateCustomer(ctx, safeID, "Safe Corp", 10_000_000, "USD"))

	evilName := `'; DROP TABLE customers; --`
	evilID := uuid.New()
	require.NoError(t, svc.CreateCustomer(ctx, evilID, evilName, 10_000_000, "USD"))

	pathPayloads := []struct {
		name string
		path string
		want int
	}{
		{
			name: "sql_in_uuid_path",
			path: "/api/v1/customers/550e8400-e29b-41d4-a716-446655440000'%20OR%20'1'='1",
			want: http.StatusBadRequest,
		},
		{
			name: "drop_table_uuid_path",
			path: "/api/v1/customers/" + strings.ReplaceAll("'; DROP TABLE customers; --", " ", "%20"),
			want: http.StatusBadRequest,
		},
		{
			name: "union_select_uuid_path",
			path: "/api/v1/customers/00000000-0000-0000-0000-000000000000%20UNION%20SELECT%20null",
			want: http.StatusBadRequest,
		},
	}

	for _, tc := range pathPayloads {
		t.Run(tc.name, func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodGet, tc.path, http.NoBody)
			withSessionUser(req, tokenMaker, RoleAdmin, uuid.Nil)
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)
			assert.Equal(t, tc.want, w.Code, "body=%s", w.Body.String())
		})
	}

	listPayloads := []struct {
		name  string
		query string
	}{
		{name: "limit_sql_injection", query: "limit=1;DROP TABLE customers"},
		{name: "offset_or_injection", query: "offset=0' OR '1'='1"},
		{name: "limit_union", query: "limit=50 UNION SELECT null"},
	}

	for _, tc := range listPayloads {
		t.Run(tc.name, func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodGet, "/api/v1/customers?"+tc.query, http.NoBody)
			withSessionUser(req, tokenMaker, RoleAdmin, uuid.Nil)
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)
			require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())

			var resp CustomerListResponse
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
			assert.GreaterOrEqual(t, resp.Total, int64(2))
		})
	}

	t.Run("malicious_name_stored_and_returned", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/customers/"+evilID.String(), http.NoBody)
		withSessionUser(req, tokenMaker, RoleAdmin, uuid.Nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code)

		var dto CustomerDTO
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dto))
		assert.Equal(t, evilName, dto.Name)
	})

	var count int
	require.NoError(t, pool.QueryRow(ctx, `SELECT COUNT(*) FROM customers`).Scan(&count))
	assert.GreaterOrEqual(t, count, 2)

	var storedName string
	require.NoError(t, pool.QueryRow(ctx, `SELECT name FROM customers WHERE id = $1`, evilID).Scan(&storedName))
	assert.Equal(t, evilName, storedName)

	require.NoError(t, pool.QueryRow(ctx, `SELECT COUNT(*) FROM customers WHERE id = $1`, safeID).Scan(&count))
	assert.Equal(t, 1, count)
}
