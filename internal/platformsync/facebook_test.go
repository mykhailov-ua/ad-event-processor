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

func TestFetchFacebookCampaignStatus_httptest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/12345", r.URL.Path)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status":       "ACTIVE",
			"daily_budget": "10000",
		})
	}))
	defer srv.Close()

	status, err := platformsync.FetchFacebookCampaignStatusForTest(context.Background(), srv.Client(), srv.URL, costsync.Credential{
		AccessToken: "tok",
	}, "12345")
	require.NoError(t, err)
	require.Equal(t, "ACTIVE", status.Status)
	require.True(t, status.HasDailyBudgetMicro)
	require.Equal(t, int64(100_000_000), status.DailyBudgetMicro)
}

func TestMutateFacebookCampaign_pause_httptest(t *testing.T) {
	var posted bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.NoError(t, r.ParseForm())
		require.Equal(t, "PAUSED", r.Form.Get("status"))
		posted = true
		_, _ = w.Write([]byte(`{"success":true}`))
	}))
	defer srv.Close()

	_, err := platformsync.MutateFacebookCampaignForTest(context.Background(), srv.Client(), srv.URL, costsync.Credential{
		AccessToken: "tok",
	}, "12345", platformsync.ActionPause, platformsync.MutationRequest{})
	require.NoError(t, err)
	require.True(t, posted)
}
