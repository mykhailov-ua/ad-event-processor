package dashboardadmin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ad-event-processor/internal/reports"
)

type stubBuyerPortfolio struct {
	called    uuid.UUID
	portfolio BuyerPortfolioDTO
	err       error
}

func (s *stubBuyerPortfolio) GetBuyerPortfolio(_ context.Context, customerID uuid.UUID) (BuyerPortfolioDTO, error) {
	s.called = customerID
	if s.err != nil {
		return BuyerPortfolioDTO{}, s.err
	}
	return s.portfolio, nil
}

func (s *stubBuyerPortfolio) GetBuyerPortfolioRange(ctx context.Context, customerID uuid.UUID, _ *uuid.UUID, _, _ time.Time, _ reports.ChartGranularity) (BuyerPortfolioDTO, error) {
	return s.GetBuyerPortfolio(ctx, customerID)
}

func (s *stubBuyerPortfolio) GetBuyerDrilldown(_ context.Context, _ uuid.UUID, _ uuid.UUID, _, _ time.Time, _ reports.DashboardDrilldownFilter) (reports.DashboardBreakdownTableDTO, error) {
	return reports.DashboardBreakdownTableDTO{Rows: []reports.DashboardBreakdownRowDTO{}}, nil
}

func TestGetBuyerDashboard_OK(t *testing.T) {
	t.Parallel()
	custID := uuid.New()
	stub := &stubBuyerPortfolio{
		portfolio: BuyerPortfolioDTO{
			CustomerID:    custID.String(),
			Active:        2,
			Paused:        1,
			Impressions7d: 1200,
			Clicks7d:      45,
			Campaigns: []BuyerCampaignPortfolioRowDTO{
				{ID: uuid.New().String(), Name: "A", Status: "ACTIVE", Impressions7d: 800, Clicks7d: 30},
			},
		},
	}
	h := &HTTPHandlers{
		BuyerPortfolio: stub,
		ResolveCustomerID: func(_ *http.Request, body *uuid.UUID) (uuid.UUID, error) {
			if body != nil && *body != uuid.Nil {
				return *body, nil
			}
			return custID, nil
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboards/buyer", http.NoBody)
	rec := httptest.NewRecorder()
	h.getBuyerDashboard(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, custID, stub.called)

	var resp BuyerPortfolioDTO
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, 2, resp.Active)
	assert.Equal(t, int64(1200), resp.Impressions7d)
	require.Len(t, resp.Campaigns, 1)
}

func TestGetBuyerDashboard_includesFraudOverview(t *testing.T) {
	t.Parallel()
	custID := uuid.New()
	stub := &stubBuyerPortfolio{
		portfolio: BuyerPortfolioDTO{
			CustomerID: custID.String(),
			Fraud: &CustomerFraudOverviewDTO{
				TotalEvents:      100,
				BlockRateDisplay: "25.0%",
				Freshness:        DataFreshnessDTO{Stale: false},
			},
		},
	}
	h := &HTTPHandlers{
		BuyerPortfolio: stub,
		ResolveCustomerID: func(_ *http.Request, body *uuid.UUID) (uuid.UUID, error) {
			return custID, nil
		},
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboards/buyer", http.NoBody)
	rec := httptest.NewRecorder()
	h.getBuyerDashboard(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	var resp BuyerPortfolioDTO
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.NotNil(t, resp.Fraud)
	assert.Equal(t, "25.0%", resp.Fraud.BlockRateDisplay)
}

func TestGetBuyerDashboard_requiresCustomerID(t *testing.T) {
	t.Parallel()
	h := &HTTPHandlers{
		BuyerPortfolio: &stubBuyerPortfolio{},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboards/buyer", http.NoBody)
	rec := httptest.NewRecorder()
	h.getBuyerDashboard(rec, req)
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestGetBuyerDrilldown_requiresCampaignAndDimension(t *testing.T) {
	t.Parallel()
	custID := uuid.New()
	h := &HTTPHandlers{
		BuyerPortfolio: &stubBuyerPortfolio{},
		ResolveCustomerID: func(_ *http.Request, _ *uuid.UUID) (uuid.UUID, error) {
			return custID, nil
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboards/buyer/drilldown?customer_id="+custID.String(), http.NoBody)
	rec := httptest.NewRecorder()
	h.getBuyerDrilldown(rec, req)
	require.Equal(t, http.StatusBadRequest, rec.Code)

	req = httptest.NewRequest(
		http.MethodGet,
		"/api/v1/dashboards/buyer/drilldown?customer_id="+custID.String()+"&campaign_id="+uuid.NewString()+"&dimension=bad",
		http.NoBody,
	)
	rec = httptest.NewRecorder()
	h.getBuyerDrilldown(rec, req)
	require.Equal(t, http.StatusBadRequest, rec.Code)
}
