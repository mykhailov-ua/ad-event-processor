package campaign

import (
	"context"
	"net/http"
	"strings"

	"ad-event-processor/pkg/branding"
)

func followCampaignSmokeRedirects(ctx context.Context, client *http.Client, startURL string, maxHops int) ([]CampaignSmokeRedirectHop, string, string, error) {
	chain := make([]CampaignSmokeRedirectHop, 0, maxHops+1)
	current := startURL
	for hop := 0; hop <= maxHops; hop++ {
		hopChain, outcome, finalHost, err, done := followCampaignSmokeHop(ctx, client, current, hop)
		chain = append(chain, hopChain...)
		if done {
			return chain, outcome, finalHost, err
		}
		current = finalHost
	}
	return chain, "redirect_limit_exceeded", hostFromURL(current), nil
}

func followCampaignSmokeHop(ctx context.Context, client *http.Client, current string, hop int) (chain []CampaignSmokeRedirectHop, outcome, nextURL string, err error, done bool) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, current, http.NoBody)
	if err != nil {
		return nil, "tracker_unreachable", "", err, true
	}
	req.Header.Set("User-Agent", branding.HTTPUserAgent("CampaignSmoke"))
	resp, err := client.Do(req)
	if err != nil {
		return nil, "tracker_unreachable", "", err, true
	}
	defer drainHTTPBody(resp.Body)

	chain = []CampaignSmokeRedirectHop{{URL: current, StatusCode: resp.StatusCode}}
	if hop == 0 && resp.StatusCode == http.StatusForbidden {
		return chain, "click_rejected", hostFromURL(current), nil, true
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		return chain, "click_rejected", hostFromURL(current), nil, true
	}
	if resp.StatusCode >= 300 && resp.StatusCode < 400 {
		location := strings.TrimSpace(resp.Header.Get("Location"))
		if location == "" {
			return chain, "redirect_missing_location", hostFromURL(current), nil, true
		}
		next, err := resolveCampaignSmokeRedirect(current, location)
		if err != nil {
			return chain, "redirect_invalid_location", hostFromURL(current), nil, true
		}
		return chain, "", next, nil, false
	}
	finalHost := hostFromURL(current)
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return chain, "", finalHost, nil, true
	}
	return chain, "non_success_status", finalHost, nil, true
}
