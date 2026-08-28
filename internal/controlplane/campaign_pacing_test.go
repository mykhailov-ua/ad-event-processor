package controlplane

import (
	ctrlhttp "ad-event-processor/internal/control/http"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"ad-event-processor/internal/campaign"
	"ad-event-processor/internal/config"
	"ad-event-processor/internal/database"
	"ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestManagementAPI_CampaignPacing(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}

	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()

	redisClient, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	cfg := &config.Config{
		AdminAPIKey:           "test-secret-pacing",
		CampaignUpdateChannel: "test:campaign-updates-pacing",
		TokenSymmetricKey:     "01234567890123456789012345678901",
	}

	authMW, tokenMaker := integrationTestAuth(t, redisClient, cfg)
	svc := NewService(context.Background(), pool, []redis.UniversalClient{redisClient}, nil, cfg)
	defer svc.Close()
	h := NewHandler(svc, cfg, authMW, nil, nil, nil)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	ctx := context.Background()
	custID := uuid.New()
	err := svc.CreateCustomer(ctx, custID, "Advertiser Pacing", 500_000_000, "USD")
	require.NoError(t, err)

	campID, err := svc.CreateCampaign(ctx, campaign.CreateCampaignSpec{
		CustomerID:       custID,
		Name:             "Spring Sale Pacing",
		BudgetLimitMicro: 100_000_000,
		PacingMode:       string(db.PacingModeTypeEVEN),
		DailyBudgetMicro: 10_000_000,
		Timezone:         "UTC",
		FreqLimit:        5,
		FreqWindow:       3600,
		TargetCountries:  []string{"US", "GB"},
		IdempotencyKey:   "idemp-camp-pacing-1",
	})
	require.NoError(t, err)

	t.Run("InvalidPacingMode", func(t *testing.T) {
		_, err := svc.UpdateCampaignPacing(ctx, campID, "INVALID")
		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrInvalidPacingMode))
	})

	t.Run("CampaignIsolation_Forbidden", func(t *testing.T) {
		otherCustID := uuid.New()

		req, _ := http.NewRequest("GET", "/api/v1/campaigns/"+campID.String(), http.NoBody)
		withSessionUser(req, tokenMaker, ctrlhttp.RoleUser, otherCustID)

		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusForbidden, resp.Code)
	})

	t.Run("UpdatePacing_Success", func(t *testing.T) {
		camp, err := svc.UpdateCampaignPacing(ctx, campID, "ASAP")
		require.NoError(t, err)
		assert.Equal(t, "ASAP", camp.PacingMode)

		var currentPacing string
		err = pool.QueryRow(ctx, "SELECT pacing_mode FROM campaigns WHERE id = $1", campID).Scan(&currentPacing)
		require.NoError(t, err)
		assert.Equal(t, "ASAP", currentPacing)

		var auditCount int
		err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM admin_audit_log WHERE action = 'UPDATE_CAMPAIGN_PACING' AND target_id = $1", campID).Scan(&auditCount)
		require.NoError(t, err)
		assert.Equal(t, 1, auditCount)

		worker := NewOutboxWorker(svc)
		require.NoError(t, worker.ProcessOutbox(ctx))

		assert.Eventually(t, func() bool {
			val, rdbErr := redisClient.HGet(ctx, "campaign:settings:"+campID.String(), "pacing_mode").Result()
			return rdbErr == nil && val == "ASAP"
		}, 3*time.Second, 50*time.Millisecond)
	})

	t.Run("UpdatePacing_VPP_and_off", func(t *testing.T) {
		camp, err := svc.UpdateCampaignPacing(ctx, campID, "vpp")
		require.NoError(t, err)
		assert.Equal(t, "VPP", camp.PacingMode)

		camp, err = svc.UpdateCampaignPacing(ctx, campID, "off")
		require.NoError(t, err)
		assert.Equal(t, "EVEN", camp.PacingMode)
	})
}
