package platformsync

import (
	"context"
	"net/http"

	"ad-event-processor/internal/costsync"
)

func FetchFacebookCampaignStatusForTest(ctx context.Context, client *http.Client, baseURL string, cred costsync.Credential, externalCampaignID string) (RemoteCampaignStatus, error) {
	return fetchFacebookCampaignStatus(ctx, client, baseURL, cred, externalCampaignID)
}

func MutateFacebookCampaignForTest(ctx context.Context, client *http.Client, baseURL string, cred costsync.Credential, externalCampaignID, action string, req MutationRequest) (map[string]string, error) {
	return mutateFacebookCampaign(ctx, client, baseURL, cred, externalCampaignID, action, req)
}
