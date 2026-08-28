package reports

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReports_Placements(t *testing.T) {
	t.Parallel()

	h := &ReportsHTTPHandlers{}
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest("GET", "/api/v1/reports/placements?customer_id="+uuid.New().String()+"&limit=5", http.NoBody)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	require.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestReports_Keywords(t *testing.T) {
	t.Parallel()

	h := &ReportsHTTPHandlers{}
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest("GET", "/api/v1/reports/keywords?customer_id="+uuid.New().String()+"&limit=5", http.NoBody)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	require.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestToPlacementReportRowDTO(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		row  reportMetricsCHRow
		want PlacementReportRowDTO
	}{
		{
			name: "profit and roi",
			row: reportMetricsCHRow{
				Dimension:    "zone_1001",
				CampaignID:   "camp-1",
				Impressions:  10000,
				Clicks:       500,
				Conversions:  10,
				SpendMicro:   50_000_000,
				RevenueMicro: 60_000_000,
			},
			want: PlacementReportRowDTO{
				PlacementID:  "zone_1001",
				CampaignID:   "camp-1",
				Impressions:  10000,
				Clicks:       500,
				Conversions:  10,
				SpendMicro:   50_000_000,
				RevenueMicro: 60_000_000,
				ProfitMicro:  10_000_000,
				ROIPct:       20,
				CPAMicro:     5_000_000,
				CTR:          0.05,
			},
		},
		{
			name: "zero spend skips roi",
			row: reportMetricsCHRow{
				Dimension:  "zone_0",
				CampaignID: "camp-2",
			},
			want: PlacementReportRowDTO{
				PlacementID: "zone_0",
				CampaignID:  "camp-2",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ToPlacementReportRowDTO(tt.row, 0)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestToKeywordReportRowDTO(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		row  reportMetricsCHRow
		want KeywordReportRowDTO
	}{
		{
			name: "profit and roi",
			row: reportMetricsCHRow{
				Dimension:    "insurance",
				CampaignID:   "camp-1",
				Impressions:  5000,
				Clicks:       200,
				Conversions:  5,
				SpendMicro:   25_000_000,
				RevenueMicro: 30_000_000,
			},
			want: KeywordReportRowDTO{
				Keyword:      "insurance",
				CampaignID:   "camp-1",
				Impressions:  5000,
				Clicks:       200,
				Conversions:  5,
				SpendMicro:   25_000_000,
				RevenueMicro: 30_000_000,
				ProfitMicro:  5_000_000,
				ROIPct:       20,
				CPAMicro:     5_000_000,
				CTR:          0.04,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toKeywordReportRowDTO(tt.row, 0)
			assert.Equal(t, tt.want, got)
		})
	}
}
