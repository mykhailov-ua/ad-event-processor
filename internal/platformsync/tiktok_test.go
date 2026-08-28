package platformsync_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"ad-event-processor/internal/costsync"
	"ad-event-processor/internal/platformsync"

	"github.com/stretchr/testify/require"
)

func TestFetchTikTokCampaignStatus_httptest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/open_api/v1.3/campaign/get/", r.URL.Path)
		require.Equal(t, "tok", r.Header.Get("Access-Token"))
		require.Equal(t, "adv-1", r.URL.Query().Get("advertiser_id"))
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"list": []map[string]string{{
					"campaign_id":      "camp-9",
					"operation_status": "ENABLE",
					"budget_mode":      "BUDGET_MODE_DAY",
					"budget":           "12.50",
				}},
			},
		})
	}))
	defer srv.Close()

	status, err := platformsync.FetchTikTokCampaignStatusForTest(context.Background(), srv.Client(), srv.URL+"/open_api/v1.3", costsync.Credential{
		AccountID:   "adv-1",
		AccessToken: "tok",
	}, "camp-9")
	require.NoError(t, err)
	require.Equal(t, "ACTIVE", status.Status)
	require.True(t, status.HasDailyBudgetMicro)
	require.Equal(t, int64(12_500_000), status.DailyBudgetMicro)
}

func TestMutateTikTokCampaign_pause_httptest(t *testing.T) {
	var posted bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/open_api/v1.3/campaign/status/update/", r.URL.Path)
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "tok", r.Header.Get("Access-Token"))
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		require.Equal(t, "adv-1", body["advertiser_id"])
		require.Equal(t, "DISABLE", body["operation_status"])
		posted = true
		_, _ = w.Write([]byte(`{"code":0}`))
	}))
	defer srv.Close()

	_, err := platformsync.MutateTikTokCampaignForTest(context.Background(), srv.Client(), srv.URL+"/open_api/v1.3", costsync.Credential{
		AccountID:   "adv-1",
		AccessToken: "tok",
	}, "camp-9", platformsync.ActionPause, platformsync.MutationRequest{})
	require.NoError(t, err)
	require.True(t, posted)
}

func TestMutateTikTokCampaign_pauseFailure_holdout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "upstream failure", http.StatusInternalServerError)
	}))
	defer srv.Close()

	_, err := platformsync.MutateTikTokCampaignForTest(context.Background(), srv.Client(), srv.URL+"/open_api/v1.3", costsync.Credential{
		AccountID:   "adv-1",
		AccessToken: "tok",
	}, "camp-9", platformsync.ActionPause, platformsync.MutationRequest{})
	require.Error(t, err)
}
