package controlplane

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	ctrlhttp "ad-event-processor/internal/control/http"

	"ad-event-processor/internal/campaign"
	"ad-event-processor/internal/config"
	"ad-event-processor/internal/database"
	"ad-event-processor/internal/identity"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

const (
	queryBudgetSublinearExportMaxPerItem = 1
	// Per-row pause/resume/archive still issues O(N) UPDATE/history/outbox in one txn; slope ~3-4 is expected.
	// N separate transactions (pre batch fix) produced slope ~5-6 (GetCampaignForUpdate + BEGIN per id).
	queryBudgetSublinearBulkPauseMaxItem = 4
)

type queryBudgetHarness struct {
	t          *testing.T
	counter    *database.QueryCounter
	mux        *http.ServeMux
	tokenMaker identity.Maker
	svc        *Service
	custID     uuid.UUID
}

func newQueryBudgetHarness(t *testing.T) *queryBudgetHarness {
	t.Helper()
	pool, counter, cleanup := database.SetupTestDBWithQueryCounter(t)
	t.Cleanup(cleanup)
	redisClient, cleanupRedis := database.SetupTestRedis(t)
	t.Cleanup(cleanupRedis)

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
	custID := uuid.New()
	require.NoError(t, svc.CreateCustomer(ctx, custID, "Query Budget Sublinear Corp", 500_000_000, "USD"))

	return &queryBudgetHarness{
		t:          t,
		counter:    counter,
		mux:        mux,
		tokenMaker: tokenMaker,
		svc:        svc,
		custID:     custID,
	}
}

func (h *queryBudgetHarness) createActiveCampaigns(n int, prefix string) []string {
	h.t.Helper()
	ctx := context.Background()
	ids := make([]string, 0, n)
	for i := 0; i < n; i++ {
		campID, err := h.svc.CreateCampaign(ctx, campaign.CreateCampaignSpec{
			CustomerID:       h.custID,
			Name:             fmt.Sprintf("%s %d", prefix, i),
			BudgetLimitMicro: 10_000_000,
			PacingMode:       "ASAP",
			Timezone:         "UTC",
			IdempotencyKey:   fmt.Sprintf("%s-%d-%s", prefix, i, uuid.New().String()[:8]),
		})
		require.NoError(h.t, err)
		ids = append(ids, campID.String())
	}
	return ids
}

func TestQueryBudget_Sublinear_BulkEndpoints_HTTP(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run bash scripts/ci/query_budget_gate.sh (Docker testcontainers)")
	}

	h := newQueryBudgetHarness(t)

	t.Run("export_batch", func(t *testing.T) {
		sizes := []int{2, 10}
		points := make([]database.QueryBudgetPoint, 0, len(sizes))
		for _, n := range sizes {
			ids := h.createActiveCampaigns(n, "export-sublinear")
			queries := database.MeasureQueryBudget(h.counter, func() {
				req, _ := http.NewRequest(http.MethodGet, "/api/v1/campaigns/export?ids="+strings.Join(ids, ","), http.NoBody)
				withSessionUser(req, h.tokenMaker, ctrlhttp.RoleAdmin, uuid.Nil)
				w := httptest.NewRecorder()
				h.mux.ServeHTTP(w, req)
				require.Equal(t, http.StatusOK, w.Code, w.Body.String())
			})
			points = append(points, database.QueryBudgetPoint{N: n, Queries: queries})
		}
		database.AssertQueryBudgetSublinear(t, "export_batch", points, queryBudgetSublinearExportMaxPerItem)
	})

	t.Run("bulk_pause", func(t *testing.T) {
		sizes := []int{2, 10}
		points := make([]database.QueryBudgetPoint, 0, len(sizes))
		for _, n := range sizes {
			ids := h.createActiveCampaigns(n, "pause-sublinear")
			body, err := json.Marshal(map[string]any{
				"action":       "pause",
				"campaign_ids": ids,
			})
			require.NoError(t, err)
			queries := database.MeasureQueryBudget(h.counter, func() {
				req, _ := http.NewRequest(http.MethodPost, "/api/v1/campaigns/bulk-action", bytes.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				withSessionUser(req, h.tokenMaker, ctrlhttp.RoleAdmin, uuid.Nil)
				w := httptest.NewRecorder()
				h.mux.ServeHTTP(w, req)
				require.Equal(t, http.StatusOK, w.Code, w.Body.String())
			})
			points = append(points, database.QueryBudgetPoint{N: n, Queries: queries})
		}
		database.AssertQueryBudgetSublinear(t, "bulk_pause", points, queryBudgetSublinearBulkPauseMaxItem)
	})
}
