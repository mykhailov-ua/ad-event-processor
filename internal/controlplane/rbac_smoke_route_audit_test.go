package controlplane

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"ad-event-processor/internal/config"
	ctrlhttp "ad-event-processor/internal/control/http"
	"ad-event-processor/internal/database"
	"ad-event-processor/internal/identity"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mediaBuyerSmokePrimaryGET mirrors web/e2e/helpers.js ADMIN_SMOKE_ROUTE_READS + OPS_SECTION_READS.
// Scope: smoke-matrix primary list GETs for MB (not full NAV_GROUPS / EXTRA_ROUTE_RULES).
type mediaBuyerSmokePrimaryGET struct {
	name          string
	method        string
	path          string
	wantForbidden bool
}

var mediaBuyerSmokePrimaryGETs = []mediaBuyerSmokePrimaryGET{
	{name: "customers", method: http.MethodGet, path: "/api/v1/customers", wantForbidden: false},
	{name: "campaigns", method: http.MethodGet, path: "/api/v1/campaigns", wantForbidden: false},
	{name: "billing_invoices", method: http.MethodGet, path: "/api/v1/billing/invoices", wantForbidden: false},
	{name: "settings_platform", method: http.MethodGet, path: "/api/v1/settings/platform", wantForbidden: true},
	{name: "license_status", method: http.MethodGet, path: "/api/v1/license/status", wantForbidden: false},
	{name: "team_overview", method: http.MethodGet, path: "/api/v1/team/overview", wantForbidden: false},
	{name: "audit", method: http.MethodGet, path: "/api/v1/audit", wantForbidden: true},
	{name: "reports_catalog", method: http.MethodGet, path: "/api/v1/reports/catalog", wantForbidden: false},
	{name: "ops_home", method: http.MethodGet, path: "/api/v1/ops/home", wantForbidden: true},
	{name: "fraud_presets", method: http.MethodGet, path: "/api/v1/fraud/presets", wantForbidden: false},
	{name: "ops_dlq", method: http.MethodGet, path: "/api/v1/ops/dlq/inbox", wantForbidden: true},
	{name: "ops_blacklist", method: http.MethodGet, path: "/api/v1/ops/blacklist", wantForbidden: true},
	{name: "ops_incidents", method: http.MethodGet, path: "/api/v1/ops/incidents", wantForbidden: true},
	{name: "ops_outbox", method: http.MethodGet, path: "/api/v1/ops/outbox", wantForbidden: true},
	{name: "ops_shards", method: http.MethodGet, path: "/api/v1/ops/shards", wantForbidden: true},
	{name: "ops_ml_model", method: http.MethodGet, path: "/api/v1/ops/ml-model", wantForbidden: true},
	{name: "ops_domains", method: http.MethodGet, path: "/api/v1/ops/domains/rotation", wantForbidden: true},
	{name: "ops_recon", method: http.MethodGet, path: "/api/v1/ops/recon", wantForbidden: true},
	{name: "ops_consent", method: http.MethodGet, path: "/api/v1/ops/consent/proofs", wantForbidden: true},
	{name: "ops_rum", method: http.MethodGet, path: "/api/v1/ops/rum", wantForbidden: true},
	{name: "ops_metrics", method: http.MethodGet, path: "/api/v1/ops/dashboard/metrics", wantForbidden: true},
}

func TestManagementAPI_RoleMediaBuyerSmokePrimaryGET_holdout(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: media buyer smoke-matrix primary GET RBAC audit")
	}

	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()
	redisClient, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	cfg := &config.Config{TokenSymmetricKey: "01234567890123456789012345678901"}
	tokenMaker, err := identity.NewPasetoMaker(string(cfg.TokenSymmetricKey))
	require.NoError(t, err)

	authMdl := NewAuthMiddleware(tokenMaker, redisClient, cfg, nil)
	svc := newBareService(t, pool, []redis.UniversalClient{redisClient}, cfg)
	h := NewHandler(svc, cfg, authMdl, nil, nil, nil)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	token, err := tokenMaker.CreateToken(uuid.New(), uuid.New(), ctrlhttp.RoleMediaBuyer, uuid.New(), time.Hour)
	require.NoError(t, err)

	for _, tc := range mediaBuyerSmokePrimaryGETs {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(tc.method, tc.path, http.NoBody)
			require.NoError(t, err)
			req.AddCookie(&http.Cookie{Name: "accessToken", Value: token})
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)
			if tc.wantForbidden {
				assert.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())
				assert.Contains(t, rec.Body.String(), "FORBIDDEN")
				return
			}
			assert.NotEqual(t, http.StatusForbidden, rec.Code, "RBAC leak: MB must not receive 403 on %s", tc.path)
		})
	}
}
