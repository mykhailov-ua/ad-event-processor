package adminapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	h := &DashboardsHTTPHandlers{
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

func TestGetBuyerDashboard_requiresCustomerID(t *testing.T) {
	t.Parallel()
	h := &DashboardsHTTPHandlers{
		BuyerPortfolio: &stubBuyerPortfolio{},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboards/buyer", http.NoBody)
	rec := httptest.NewRecorder()
	h.getBuyerDashboard(rec, req)
	require.Equal(t, http.StatusBadRequest, rec.Code)
}
