package costsync

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

func fetchNetworkCosts(ctx context.Context, client *http.Client, network, baseURL string, cred Credential, date time.Time) ([]CostLine, error) {
	switch network {
	case "facebook":
		return fetchFacebookCosts(ctx, client, baseURL, cred, date)
	case "taboola":
		return fetchTaboolaCosts(ctx, client, baseURL, cred, date)
	case "outbrain":
		return fetchOutbrainCosts(ctx, client, baseURL, cred, date)
	case "google":
		return fetchGoogleCosts(ctx, client, baseURL, cred, date)
	case "tonic_rsoc":
		return fetchTonicRSOCCosts(ctx, client, baseURL, cred, date)
	case "system1_rsoc":
		return fetchSystem1RSOCCosts(ctx, client, baseURL, cred, date)
	case "tiktok":
		return fetchTikTokCosts(ctx, client, baseURL, cred, date)
	case "propellerads":
		return fetchPropellerAdsCosts(ctx, client, baseURL, cred, date)
	case "mgid":
		return fetchMGIDCosts(ctx, client, baseURL, cred, date)
	case "adsterra":
		return fetchAdsterraCosts(ctx, client, baseURL, cred, date)
	case "exoclick":
		return fetchExoClickCosts(ctx, client, baseURL, cred, date)
	case "hilltopads":
		return fetchHilltopAdsCosts(ctx, client, baseURL, cred, date)
	case "clickadu":
		return fetchClickaduCosts(ctx, client, baseURL, cred, date)
	case "popads":
		return fetchPopAdsCosts(ctx, client, baseURL, cred, date)
	case "revcontent":
		return fetchRevcontentCosts(ctx, client, baseURL, cred, date)
	case "microsoft_ads":
		return fetchMicrosoftAdsCosts(ctx, client, baseURL, cred, date)
	case "snapchat":
		return fetchSnapchatCosts(ctx, client, baseURL, cred, date)
	case "linkedin":
		return fetchLinkedInCosts(ctx, client, baseURL, cred, date)
	case "pinterest":
		return fetchPinterestCosts(ctx, client, baseURL, cred, date)
	case "trafficstars":
		return fetchTrafficStarsCosts(ctx, client, baseURL, cred, date)
	case "richads":
		return fetchRichAdsCosts(ctx, client, baseURL, cred, date)
	case "galaksion":
		return fetchGalaksionCosts(ctx, client, baseURL, cred, date)
	default:
		return nil, fmt.Errorf("unsupported network: %s", network)
	}
}
