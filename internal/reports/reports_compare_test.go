package reports

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPreviousReportRange(t *testing.T) {
	t.Parallel()
	from := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 8, 8, 0, 0, 0, 0, time.UTC)
	prevFrom, prevTo := previousReportRange(from, to)
	assert.Equal(t, time.Date(2026, 7, 25, 0, 0, 0, 0, time.UTC), prevFrom)
	assert.Equal(t, from, prevTo)
}

func TestAttachPlacementCompareDeltas(t *testing.T) {
	t.Parallel()
	rows := []PlacementReportRowDTO{{
		PlacementID: "p1", CampaignID: "c1",
		SpendMicro: 200, Clicks: 20, Impressions: 100,
	}}
	prev := []reportMetricsCHRow{{
		Dimension: "p1", CampaignID: "c1",
		SpendMicro: 150, Clicks: 10, Impressions: 80,
	}}
	attachPlacementCompareDeltas(rows, prev)
	if assert.NotNil(t, rows[0].Compare) {
		assert.Equal(t, int64(50), rows[0].Compare.SpendMicroDelta)
		assert.Equal(t, int64(10), rows[0].Compare.ClicksDelta)
	}
}

func TestAttachTrafficCompareDeltas(t *testing.T) {
	t.Parallel()
	rows := []TrafficSourceRowDTO{{
		Channel: "paid_search", SpendMicro: 200, Clicks: 20,
	}}
	prev := []TrafficSourceRowDTO{{
		Channel: "paid_search", SpendMicro: 150, Clicks: 10,
	}}
	attachTrafficCompareDeltas(rows, prev)
	if assert.NotNil(t, rows[0].Compare) {
		assert.Equal(t, int64(50), rows[0].Compare.SpendMicroDelta)
	}
}

func TestAttachMapCompareDeltas(t *testing.T) {
	t.Parallel()
	rows := []map[string]any{{
		"campaign_id": "c1", "ad_spend_micro": int64(300), "clicks": int64(30),
	}}
	prev := []map[string]any{{
		"campaign_id": "c1", "ad_spend_micro": int64(100), "clicks": int64(10),
	}}
	attachMapCompareDeltas(rows, prev, "campaign_id")
	compare, ok := rows[0]["compare"].(map[string]any)
	if assert.True(t, ok) {
		assert.Equal(t, int64(200), compare["spend_micro_delta"])
		assert.Equal(t, int64(20), compare["clicks_delta"])
	}
}

func TestParseComparePrevious(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest("GET", "/api/v1/reports/placements?compare=previous", http.NoBody)
	assert.True(t, parseComparePrevious(req))
	req2 := httptest.NewRequest("GET", "/api/v1/reports/placements?compare_period=true", http.NoBody)
	assert.True(t, parseComparePrevious(req2))
}
