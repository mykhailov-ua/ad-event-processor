package controlplane

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bidshard/ad-event-processor/internal/config"
	"github.com/bidshard/ad-event-processor/internal/controlplane/adminapi"
	"github.com/bidshard/ad-event-processor/internal/database"
	"github.com/bidshard/ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestManagementAPI_Campaigns(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()

	rdb, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	cfg := &config.Config{
		AdminAPIKey:       "test-secret",
		TokenSymmetricKey: "01234567890123456789012345678901",
	}

	authMW, tokenMaker := integrationTestAuth(t, rdb, cfg)
	svc := NewService(context.Background(), pool, []redis.UniversalClient{rdb}, nil, cfg)
	defer svc.Close()
	h := NewHandler(svc, cfg, authMW, nil, nil, nil)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	ctx := context.Background()
	custID := uuid.New()
	err := svc.CreateCustomer(ctx, custID, "Advertiser", 500_000_000, "USD")
	require.NoError(t, err)

	campID, err := svc.CreateCampaign(ctx, CampaignCreateSpec{
		CustomerID:       custID,
		Name:             "Spring Sale",
		BudgetLimitMicro: 100_000_000,
		PacingMode:       string(db.PacingModeTypeEVEN),
		DailyBudgetMicro: 10_000_000,
		Timezone:         "UTC",
		FreqLimit:        5,
		FreqWindow:       3600,
		TargetCountries:  []string{"US", "GB"},
		IdempotencyKey:   "idemp-camp-1",
	})
	require.NoError(t, err)

	t.Run("ListCampaigns", func(t *testing.T) {
		campaigns, total, err := svc.ListCampaigns(ctx, custID, "ACTIVE", 50, 0)
		require.NoError(t, err)
		assert.Greater(t, total, int64(0))
		require.NotEmpty(t, campaigns)

		var found *CampaignDTO
		for i := range campaigns {
			if campaigns[i].ID == campID.String() {
				found = &campaigns[i]
				break
			}
		}
		require.NotNil(t, found)
		assert.Equal(t, "Spring Sale", found.Name)
		assert.Equal(t, "100.00", found.BudgetLimit)
		assert.Equal(t, "0.00", found.CurrentSpend)
		assert.Equal(t, []string{"US", "GB"}, found.TargetCountries)
	})

	t.Run("GetCampaignByID", func(t *testing.T) {
		camp, err := svc.GetCampaignRow(ctx, campID)
		require.NoError(t, err)
		assert.Equal(t, campID.String(), uuid.UUID(camp.ID.Bytes).String())
		assert.Equal(t, int64(100_000_000), camp.BudgetLimit)
		assert.NotNil(t, camp.TargetCountries)
	})

	t.Run("GetCampaignHistory", func(t *testing.T) {
		history, total, err := svc.ListStatusHistory(ctx, campID, 50, 0)
		require.NoError(t, err)
		assert.Greater(t, total, int64(0))
		require.NotEmpty(t, history)
		assert.Equal(t, "ACTIVE", history[0].NewStatus)
	})

	t.Run("PatchCampaign", func(t *testing.T) {
		name := "Updated Spring Sale"
		dailyMicro := int64(25_000_000)
		targetURL := "https://new.example/landing"
		countries := []string{"US", "CA"}
		freqLimit := int32(10)
		freqWindow := int32(7200)

		updated, err := svc.PatchCampaign(ctx, campID, adminapi.PatchCampaignRequest{
			Name:             &name,
			DailyBudgetMicro: &dailyMicro,
			TargetURL:        &targetURL,
			TargetCountries:  countries,
			FreqLimit:        &freqLimit,
			FreqWindow:       &freqWindow,
		})
		require.NoError(t, err)
		assert.Equal(t, name, updated.Name)
		assert.Equal(t, countries, updated.TargetCountries)
		assert.Equal(t, targetURL, updated.TargetURL)
		assert.Equal(t, int32(10), updated.FreqLimit)
		assert.Equal(t, int32(7200), updated.FreqWindow)

		body := `{"daily_budget_micro":30000000,"target_countries":["US"]}`
		req, err := http.NewRequest(http.MethodPatch, "/api/v1/campaigns/"+campID.String(), strings.NewReader(body))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		withSessionUser(req, tokenMaker, RoleAdmin, uuid.Nil)

		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)
		assert.Equal(t, http.StatusOK, resp.Code)
		assert.Contains(t, resp.Body.String(), `"daily_budget"`)
	})

	t.Run("PatchCampaign_EmptyNameRejected", func(t *testing.T) {
		empty := "   "
		_, err := svc.PatchCampaign(ctx, campID, adminapi.PatchCampaignRequest{Name: &empty})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "name is required")
	})

	t.Run("CampaignIsolation_Forbidden", func(t *testing.T) {
		otherCustID := uuid.New()

		req, _ := http.NewRequest("GET", "/api/v1/campaigns/"+campID.String(), http.NoBody)
		withSessionUser(req, tokenMaker, RoleUser, otherCustID)

		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusForbidden, resp.Code)
	})

	t.Run("CancelCampaign", func(t *testing.T) {
		require.NoError(t, svc.CancelCampaign(ctx, campID, "User requested cancellation"))

		worker := NewOutboxWorker(svc)
		require.NoError(t, worker.ProcessOutbox(ctx))
		drain := NewCampaignDrainWorker(svc)
		require.NoError(t, drain.ProcessDraining(ctx))

		assert.Eventually(t, func() bool {
			var status string
			_ = pool.QueryRow(ctx, "SELECT status FROM campaigns WHERE id = $1", campID).Scan(&status)
			return status == "DELETED"
		}, 2*time.Second, 20*time.Millisecond)
	})
}
