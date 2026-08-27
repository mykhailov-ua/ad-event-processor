package controlplane

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseSourceQualityGroupBy_dedupesAndFilters(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/reports/source-quality?group_by=country,placement&group_by=campaign,invalid", http.NoBody)
	got := parseSourceQualityGroupBy(req)
	require.Equal(t, []string{"country", "placement", "campaign"}, got)
}

func TestSourceQualityNeedsDetailRows_holdout(t *testing.T) {
	require.False(t, sourceQualityNeedsDetailRows([]string{"placement", "campaign"}))
	require.True(t, sourceQualityNeedsDetailRows([]string{"country"}))
	require.True(t, sourceQualityNeedsDetailRows([]string{"placement", "device"}))
	require.True(t, sourceQualityNeedsDetailRows([]string{"city"}))
	require.True(t, sourceQualityNeedsDetailRows([]string{"sub_id"}))
}

func TestSourceQualityDetailEventQuery_usesDimensionExprs_holdout(t *testing.T) {
	require.Contains(t, sourceQualityDetailEventQuery, "placement_id")
	require.Contains(t, sourceQualityDetailEventQuery, chDimCountryExpr)
	require.Contains(t, sourceQualityDetailEventQuery, chDimCityExpr)
	require.Contains(t, sourceQualityDetailEventQuery, chDimDeviceExpr)
	require.Contains(t, sourceQualityDetailEventQuery, chDimSub1Expr)
}

func TestAllocatePlacementCampaignShare_prefersClicks(t *testing.T) {
	share := allocatePlacementCampaignShare(25, 100, 0)
	require.InDelta(t, 0.25, share, 1e-9)
	require.Equal(t, float64(0), allocatePlacementCampaignShare(0, 0, 10))
}
