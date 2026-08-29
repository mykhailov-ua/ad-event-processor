package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestFetchFacebookCosts_Httptest(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	customerID := uuid.New()
	campaignID := uuid.New()
	day := time.Date(2026, 8, 25, 0, 0, 0, 0, time.UTC)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Contains(t, r.URL.Path, "/act_12345/insights")
		require.Equal(t, "tok-test", r.URL.Query().Get("access_token"))
		require.Contains(t, r.URL.Query().Get("time_range"), "2026-08-25")
		_ = json.NewEncoder(w).Encode(fbInsightsResponse{
			Data: []struct {
				CampaignID  string `json:"campaign_id"`
				AdsetID     string `json:"adset_id"`
				AdID        string `json:"ad_id"`
				Spend       string `json:"spend"`
				DateStart   string `json:"date_start"`
				Impressions string `json:"impressions"`
			}{
				{
					CampaignID: campaignID.String(),
					AdsetID:    "set-9",
					AdID:       "ad-42",
					Spend:      "12.34",
				},
			},
		})
	}))
	defer srv.Close()

	lines, err := fetchFacebookCosts(ctx, &http.Client{
		Transport: roundTripRewriteHost(srv.URL, nil),
	}, srv.URL, Credential{
		CustomerID:  customerID,
		AccountID:   "12345",
		AccessToken: "tok-test",
	}, day)
	require.NoError(t, err)
	require.Len(t, lines, 1)
	require.Equal(t, campaignID, lines[0].CampaignID)
	require.Equal(t, "ad-42", lines[0].PlacementID)
	require.Equal(t, "set-9", lines[0].AdsetID)
	require.Equal(t, int64(12_340_000), lines[0].AmountMicro)
	require.Equal(t, "USD", lines[0].Currency)
}
