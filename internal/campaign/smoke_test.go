package campaign

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFollowCampaignSmokeRedirects_brokenLander_returnsNonSuccess(t *testing.T) {
	lander := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "missing", http.StatusNotFound)
	}))
	defer lander.Close()

	tracker := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, lander.URL, http.StatusFound)
	}))
	defer tracker.Close()

	campaignID := uuid.New()
	clickURL, err := buildCampaignSmokeClickURL(tracker.URL, campaignID, "smoke-test-click")
	require.NoError(t, err)

	client := &http.Client{
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	chain, reason, finalHost, err := followCampaignSmokeRedirects(context.Background(), client, clickURL, campaignSmokeMaxRedirects)
	require.NoError(t, err)
	assert.False(t, reason == "")
	assert.Equal(t, "non_success_status", reason)
	assert.Len(t, chain, 2)
	assert.Equal(t, http.StatusFound, chain[0].StatusCode)
	assert.Equal(t, http.StatusNotFound, chain[1].StatusCode)
	assert.Contains(t, finalHost, "127.0.0.1")
}

func TestBuildCampaignSmokeClickURL_includesSmokeToken(t *testing.T) {
	campaignID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	raw, err := buildCampaignSmokeClickURL("https://trk.example", campaignID, "smoke-abc")
	require.NoError(t, err)
	assert.Contains(t, raw, "smoke=1")
	assert.Contains(t, raw, "campaign_id=11111111-1111-1111-1111-111111111111")
	assert.Contains(t, raw, "click_id=smoke-abc")
}

func TestRunCampaignSmoke_serviceUnavailable(t *testing.T) {
	_, err := RunCampaignSmoke(context.Background(), nil, uuid.New())
	require.Error(t, err)
}

type smokeHostStub struct {
	available bool
}

func (s smokeHostStub) SmokeServiceAvailable() bool { return s.available }
func (s smokeHostStub) AuthorizeCampaignSmoke(context.Context, uuid.UUID) error {
	return nil
}
func (s smokeHostStub) TrackerPublicBaseURL() string { return "" }

func TestRunCampaignSmoke_hostUnavailable(t *testing.T) {
	_, err := RunCampaignSmoke(context.Background(), smokeHostStub{}, uuid.New())
	require.Error(t, err)
}
