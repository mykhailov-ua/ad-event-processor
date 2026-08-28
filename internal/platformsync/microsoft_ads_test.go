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

func microsoftTestCredential() costsync.Credential {
	return costsync.Credential{
		AccountID:   "111",
		AccessToken: "tok",
		ExtraConfig: map[string]string{
			"customer_id":     "222",
			"developer_token": "dev",
		},
	}
}

func TestFetchMicrosoftAdsCampaignStatus_httptest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/CampaignManagement/v13/Campaigns/GetCampaignsByIds", r.URL.Path)
		require.Equal(t, "Bearer tok", r.Header.Get("Authorization"))
		require.Equal(t, "222", r.Header.Get("CustomerId"))
		require.Equal(t, "111", r.Header.Get("CustomerAccountId"))
		_ = json.NewEncoder(w).Encode(map[string]any{
			"Campaigns": []map[string]any{{
				"Id":     int64(98765),
				"Status": "Active",
			}},
		})
	}))
	defer srv.Close()

	status, err := platformsync.FetchMicrosoftAdsCampaignStatusForTest(context.Background(), srv.Client(), srv.URL+"/CampaignManagement/v13", microsoftTestCredential(), "98765")
	require.NoError(t, err)
	require.Equal(t, "ACTIVE", status.Status)
}

func TestMutateMicrosoftAdsCampaign_pause_httptest(t *testing.T) {
	var posted bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/CampaignManagement/v13/Campaigns/UpdateCampaigns", r.URL.Path)
		require.Equal(t, http.MethodPost, r.Method)
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		campaigns, ok := body["Campaigns"].([]any)
		require.True(t, ok)
		require.Len(t, campaigns, 1)
		campaign, ok := campaigns[0].(map[string]any)
		require.True(t, ok)
		require.Equal(t, "Paused", campaign["Status"])
		posted = true
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	_, err := platformsync.MutateMicrosoftAdsCampaignForTest(context.Background(), srv.Client(), srv.URL+"/CampaignManagement/v13", microsoftTestCredential(), "98765", platformsync.ActionPause, platformsync.MutationRequest{})
	require.NoError(t, err)
	require.True(t, posted)
}

func TestMutateMicrosoftAdsCampaign_pauseFailure_holdout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "upstream failure", http.StatusBadGateway)
	}))
	defer srv.Close()

	_, err := platformsync.MutateMicrosoftAdsCampaignForTest(context.Background(), srv.Client(), srv.URL+"/CampaignManagement/v13", microsoftTestCredential(), "98765", platformsync.ActionPause, platformsync.MutationRequest{})
	require.Error(t, err)
}
