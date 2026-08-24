package controlplane

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/database"
	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/identity"

	"github.com/google/uuid"
	redis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSlotMapAPI_RBAC(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}
	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()
	rdb, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	cfg := &config.Config{
		AdminAPIKey:       "test-secret",
		TokenSymmetricKey: "01234567890123456789012345678901",
	}
	tokenMaker, err := identity.NewPasetoMaker(string(cfg.TokenSymmetricKey))
	require.NoError(t, err)

	svc := NewService(context.Background(), pool, []redis.UniversalClient{rdb}, nil, cfg)
	defer svc.Close()

	authMW := NewAuthMiddleware(tokenMaker, rdb, cfg, nil)
	h := NewHandler(svc, cfg, authMW, nil, nil, nil)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	ctx := context.Background()
	managerToken, err := tokenMaker.CreateToken(uuid.New(), uuid.New(), "manager", uuid.New(), time.Hour)
	require.NoError(t, err)

	t.Run("manager forbidden shards read", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/ops/shards", http.NoBody)
		req.AddCookie(&http.Cookie{Name: "accessToken", Value: managerToken})
		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)
		assert.Equal(t, http.StatusForbidden, resp.Code)
	})

	t.Run("admin can create version via service", func(t *testing.T) {
		version, err := svc.CreateSlotMapVersion(ctx, uuid.Nil, nil, []domain.SlotOverride{
			{Slot: 5, ShardID: 2, State: "MIGRATING"},
		})
		require.NoError(t, err)
		assert.Equal(t, int32(2), version)
	})

	t.Run("admin audit log written", func(t *testing.T) {
		var count int64
		err := pool.QueryRow(ctx,
			"SELECT COUNT(*) FROM admin_audit_log WHERE action = 'SLOT_MAP_VERSION_CREATED'",
		).Scan(&count)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, count, int64(1))
	})
}

func TestSlotMapAPI_markMigratingAndActivate(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}
	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()
	rdb, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	cfg := &config.Config{AdminAPIKey: "test-secret"}
	rdbs := []redis.UniversalClient{rdb, rdb, rdb, rdb}
	svc := NewService(context.Background(), pool, rdbs, nil, cfg)
	defer svc.Close()

	ctx := context.Background()

	version, err := svc.CreateSlotMapVersion(ctx, uuid.Nil, nil, nil)
	require.NoError(t, err)
	require.Equal(t, int32(2), version)

	require.NoError(t, svc.MarkSlotMapMigrating(ctx, uuid.Nil, version, []int16{100, 101}, 3))
	require.NoError(t, svc.CopyAllMigratingSlots(ctx, version))
	require.NoError(t, svc.ActivateSlotMapVersion(ctx, uuid.Nil, version))

	var active int32
	err = pool.QueryRow(ctx, "SELECT active_version FROM redis_slot_map_meta WHERE id = 1").Scan(&active)
	require.NoError(t, err)
	assert.Equal(t, int32(2), active)
}

func TestHasPermission_shardsRBAC(t *testing.T) {
	assert.True(t, HasPermission(RoleAdmin, PermShardsWrite))
	assert.True(t, HasPermission(RoleAdmin, PermShardsRead))
	assert.False(t, HasPermission(RoleManager, PermShardsWrite))
	assert.False(t, HasPermission(RoleUser, PermShardsRead))
}
