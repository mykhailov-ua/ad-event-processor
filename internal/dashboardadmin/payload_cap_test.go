package dashboardadmin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCapDashboardTableSections_truncatesObjectArrays(t *testing.T) {
	t.Parallel()
	rows := make([]any, 0, 120)
	for i := 0; i < 120; i++ {
		rows = append(rows, map[string]any{"id": i})
	}
	payload := map[string]any{"campaigns": rows}
	out := capDashboardTableSections(payload)
	capped, ok := out["campaigns"].([]any)
	require.True(t, ok)
	require.Len(t, capped, maxDashboardTableRows)

	metaRaw, ok := out["table_sections_meta"].(map[string]any)
	require.True(t, ok)
	campaignsMeta, ok := metaRaw["campaigns"].(DashboardTableMeta)
	require.True(t, ok)
	require.True(t, campaignsMeta.Truncated)
	require.Equal(t, 120, campaignsMeta.Total)
}

func TestWriteRoleDashboardJSON_capsCampaignRows(t *testing.T) {
	t.Parallel()
	rows := make([]BuyerCampaignPortfolioRowDTO, 0, 105)
	for i := 0; i < 105; i++ {
		rows = append(rows, BuyerCampaignPortfolioRowDTO{ID: "c", Name: "n", Status: "ACTIVE"})
	}
	rec := httptest.NewRecorder()
	writeRoleDashboardJSON(rec, BuyerPortfolioDTO{Campaigns: rows})
	require.Equal(t, http.StatusOK, rec.Code)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
	campaigns, ok := payload["campaigns"].([]any)
	require.True(t, ok)
	require.Len(t, campaigns, maxDashboardTableRows)
}
