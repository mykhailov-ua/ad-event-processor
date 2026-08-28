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

func FetchTikTokCampaignStatusForTest(ctx context.Context, client *http.Client, baseURL string, cred costsync.Credential, externalCampaignID string) (RemoteCampaignStatus, error) {
	return fetchTikTokCampaignStatus(ctx, client, baseURL, cred, externalCampaignID)
}

func MutateTikTokCampaignForTest(ctx context.Context, client *http.Client, baseURL string, cred costsync.Credential, externalCampaignID, action string, req MutationRequest) (map[string]string, error) {
	return mutateTikTokCampaign(ctx, client, baseURL, cred, externalCampaignID, action, req)
}

func FetchMicrosoftAdsCampaignStatusForTest(ctx context.Context, client *http.Client, baseURL string, cred costsync.Credential, externalCampaignID string) (RemoteCampaignStatus, error) {
	return fetchMicrosoftAdsCampaignStatus(ctx, client, baseURL, cred, externalCampaignID)
}

func MutateMicrosoftAdsCampaignForTest(ctx context.Context, client *http.Client, baseURL string, cred costsync.Credential, externalCampaignID, action string, req MutationRequest) (map[string]string, error) {
	return mutateMicrosoftAdsCampaign(ctx, client, baseURL, cred, externalCampaignID, action, req)
}
