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
	default:
		return nil, fmt.Errorf("unsupported network: %s", network)
	}
}
