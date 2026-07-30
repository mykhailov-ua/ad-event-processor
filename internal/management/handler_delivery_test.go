package management

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"espx/internal/auth"
	"espx/internal/config"
	"espx/internal/database"
	"espx/internal/ingestion/sqlc"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestManagementAPI_DeliveryRoutes(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()
	rdb, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	cfg := &config.Config{
		AdminAPIKey:           "test-secret",
		CampaignUpdateChannel: "test:delivery-routes",
	}
	svc := newBareService(t, pool, []redis.UniversalClient{rdb}, cfg)

	ctx := context.Background()
	custID := uuid.New()
	require.NoError(t, svc.CreateCustomer(ctx, custID, "Delivery Customer", 800_000_000, "USD"))

	brandID, err := svc.CreateBrand(ctx, custID, "Delivery Brand")
	require.NoError(t, err)

	var templateID uuid.UUID
	t.Run("CreateCampaignTemplate", func(t *testing.T) {
		templateID, err = svc.CreateCampaignTemplate(
			ctx, custID, "Tpl HTTP", 50_000_000, db.PacingModeTypeEVEN, 10_000_000,
			"UTC", 0, 86400, nil, nil, []int16{9, 10},
		)
		require.NoError(t, err)
		require.NotEqual(t, uuid.Nil, templateID)
	})

	t.Run("ListCampaignTemplates", func(t *testing.T) {
		templates, total, err := svc.ListCampaignTemplates(ctx, custID, 50, 0)
		require.NoError(t, err)
		assert.Greater(t, total, int64(0))
		require.NotEmpty(t, templates)
	})

	var campID uuid.UUID
	t.Run("CreateCampaignFromTemplate", func(t *testing.T) {
		campID, err = svc.CreateCampaignFromTemplate(ctx, templateID, custID, "From Template HTTP", nil, "from-template-http")
		require.NoError(t, err)
		require.NotEqual(t, uuid.Nil, campID)
	})

	t.Run("SaveCampaignAsTemplate", func(t *testing.T) {
		savedID, err := svc.SaveCampaignAsTemplate(ctx, campID, "Saved From Camp")
		require.NoError(t, err)
		require.NotEqual(t, uuid.Nil, savedID)
	})

	t.Run("PauseAndResume", func(t *testing.T) {
		require.NoError(t, svc.PauseCampaign(ctx, campID, "manual pause"))

		var status string
		require.NoError(t, pool.QueryRow(ctx, `SELECT status::TEXT FROM campaigns WHERE id = $1`, campID).Scan(&status))
		assert.Equal(t, "PAUSED", status)

		require.NoError(t, svc.ResumeCampaign(ctx, campID, "manual resume"))

		require.NoError(t, pool.QueryRow(ctx, `SELECT status::TEXT FROM campaigns WHERE id = $1`, campID).Scan(&status))
		assert.Equal(t, "ACTIVE", status)
	})

	t.Run("UpdateCampaignSchedule", func(t *testing.T) {
		start := time.Now().Add(24 * time.Hour)
		end := time.Now().Add(72 * time.Hour)
		require.NoError(t, svc.UpdateCampaignSchedule(ctx, campID, &start, &end, []int16{8, 9}))

		camp, err := svc.GetCampaign(ctx, campID)
		require.NoError(t, err)
		assert.Equal(t, db.CampaignStatusTypePAUSED, camp.Status)
	})

	var creativeID uuid.UUID
	t.Run("BrandCreativesCRUD", func(t *testing.T) {
		creativeID, err = svc.UpsertBrandCreative(ctx, brandID, "Creative A", "https://example.com/a", 100, "ACTIVE")
		require.NoError(t, err)

		creatives, err := svc.ListBrandCreatives(ctx, brandID)
		require.NoError(t, err)
		require.NotEmpty(t, creatives)

		require.NoError(t, svc.UpdateBrandCreative(ctx, creativeID, "Creative A v2", "https://example.com/a2", 200, "ACTIVE"))
		require.NoError(t, svc.DeleteBrandCreative(ctx, creativeID))
	})
}

func TestManagementAPI_RoleUserForbiddenSettings(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()
	rdb, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	cfg := &config.Config{
		TokenSymmetricKey: "01234567890123456789012345678901",
	}
	tokenMaker, err := auth.NewPasetoMaker(string(cfg.TokenSymmetricKey))
	require.NoError(t, err)

	authMdl := NewAuthMiddleware(tokenMaker, rdb, cfg, nil)
	svc := newBareService(t, pool, []redis.UniversalClient{rdb}, cfg)
	h := NewHandler(svc, cfg, authMdl, nil, nil, nil)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	customerID := uuid.New()
	token, err := tokenMaker.CreateToken(uuid.New(), uuid.New(), "user", customerID, time.Hour)
	require.NoError(t, err)

	req, _ := http.NewRequest("POST", "/api/v1/ops/roles/reload", nil)
	req.AddCookie(&http.Cookie{Name: "accessToken", Value: token})
	resp := httptest.NewRecorder()
	mux.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusForbidden, resp.Code)
	assert.Contains(t, resp.Body.String(), "FORBIDDEN")
}

func TestManagementAPI_RoleUserForbiddenBlacklist(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()
	rdb, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	cfg := &config.Config{TokenSymmetricKey: "01234567890123456789012345678901"}
	tokenMaker, err := auth.NewPasetoMaker(string(cfg.TokenSymmetricKey))
	require.NoError(t, err)

	authMdl := NewAuthMiddleware(tokenMaker, rdb, cfg, nil)
	svc := newBareService(t, pool, []redis.UniversalClient{rdb}, cfg)
	h := NewHandler(svc, cfg, authMdl, nil, nil, nil)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	token, err := tokenMaker.CreateToken(uuid.New(), uuid.New(), "user", uuid.New(), time.Hour)
	require.NoError(t, err)

	body, _ := json.Marshal(map[string]string{"ip": "1.2.3.4", "source": "manual"})
	req, _ := http.NewRequest("POST", "/api/v1/ops/blacklist", bytes.NewReader(body))
	req.AddCookie(&http.Cookie{Name: "accessToken", Value: token})
	resp := httptest.NewRecorder()
	mux.ServeHTTP(resp, req)
	assert.Equal(t, http.StatusForbidden, resp.Code)
}

func TestManagementAPI_RoleUserForbiddenEmergencyBreaker(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()
	rdb, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	cfg := &config.Config{TokenSymmetricKey: "01234567890123456789012345678901"}
	tokenMaker, err := auth.NewPasetoMaker(string(cfg.TokenSymmetricKey))
	require.NoError(t, err)

	authMdl := NewAuthMiddleware(tokenMaker, rdb, cfg, nil)
	svc := newBareService(t, pool, []redis.UniversalClient{rdb}, cfg)
	h := NewHandler(svc, cfg, authMdl, nil, nil, nil)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	token, err := tokenMaker.CreateToken(uuid.New(), uuid.New(), "user", uuid.New(), time.Hour)
	require.NoError(t, err)

	req, _ := http.NewRequest("POST", "/api/v1/ops/roles/reload", nil)
	req.AddCookie(&http.Cookie{Name: "accessToken", Value: token})
	resp := httptest.NewRecorder()
	mux.ServeHTTP(resp, req)
	assert.Equal(t, http.StatusForbidden, resp.Code)
}
