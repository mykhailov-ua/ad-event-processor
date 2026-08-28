package campaign

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"ad-event-processor/pkg/branding"

	"github.com/google/uuid"
)

const (
	campaignSmokeMaxRedirects = 8
	campaignSmokeTimeout      = 15 * time.Second
)

var (
	errCampaignSmokeTrackerBaseMissing = errors.New("campaign smoke tracker base missing")
)

type SmokeHost interface {
	SmokeServiceAvailable() bool
	AuthorizeCampaignSmoke(ctx context.Context, campaignID uuid.UUID) error
	TrackerPublicBaseURL() string
}

func RunCampaignSmoke(ctx context.Context, host SmokeHost, campaignID uuid.UUID) (CampaignSmokeResultDTO, error) {
	if host == nil || !host.SmokeServiceAvailable() {
		return CampaignSmokeResultDTO{}, fmt.Errorf("service unavailable")
	}
	if err := host.AuthorizeCampaignSmoke(ctx, campaignID); err != nil {
		return CampaignSmokeResultDTO{}, err
	}
	base := strings.TrimSpace(host.TrackerPublicBaseURL())
	if base == "" {
		return CampaignSmokeResultDTO{}, errValidation("tracker public base URL is not configured")
	}
	clickID := "smoke-" + uuid.NewString()
	clickURL, err := buildCampaignSmokeClickURL(base, campaignID, clickID)
	if err != nil {
		return CampaignSmokeResultDTO{}, err
	}
	client := &http.Client{
		Timeout: campaignSmokeTimeout,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	result := CampaignSmokeResultDTO{CheckedAt: time.Now().UTC()}
	chain, reason, finalHost, err := followCampaignSmokeRedirects(ctx, client, clickURL, campaignSmokeMaxRedirects)
	result.RedirectChain = chain
	result.FinalHost = finalHost
	if err != nil {
		return result, err
	}
	result.FailureReason = reason
	result.Passed = reason == ""
	return result, nil
}

func buildCampaignSmokeClickURL(base string, campaignID uuid.UUID, clickID string) (string, error) {
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	if base == "" {
		return "", errCampaignSmokeTrackerBaseMissing
	}
	u, err := url.Parse(base + "/click")
	if err != nil {
		return "", err
	}
	q := u.Query()
	q.Set("campaign_id", campaignID.String())
	q.Set("click_id", clickID)
	q.Set("smoke", "1")
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func followCampaignSmokeRedirects(ctx context.Context, client *http.Client, startURL string, maxHops int) ([]CampaignSmokeRedirectHop, string, string, error) {
	chain := make([]CampaignSmokeRedirectHop, 0, maxHops+1)
	current := startURL
	for hop := 0; hop <= maxHops; hop++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, current, nil)
		if err != nil {
			return chain, "tracker_unreachable", "", err
		}
		req.Header.Set("User-Agent", branding.HTTPUserAgent("CampaignSmoke"))
		resp, err := client.Do(req)
		if err != nil {
			return chain, "tracker_unreachable", "", err
		}
		chain = append(chain, CampaignSmokeRedirectHop{URL: current, StatusCode: resp.StatusCode})
		if hop == 0 && resp.StatusCode == http.StatusForbidden {
			drainHTTPBody(resp.Body)
			return chain, "click_rejected", hostFromURL(current), nil
		}
		if resp.StatusCode == http.StatusTooManyRequests {
			drainHTTPBody(resp.Body)
			return chain, "click_rejected", hostFromURL(current), nil
		}
		if resp.StatusCode >= 300 && resp.StatusCode < 400 {
			location := strings.TrimSpace(resp.Header.Get("Location"))
			drainHTTPBody(resp.Body)
			if location == "" {
				return chain, "redirect_missing_location", hostFromURL(current), nil
			}
			next, err := resolveCampaignSmokeRedirect(current, location)
			if err != nil {
				return chain, "redirect_invalid_location", hostFromURL(current), nil
			}
			current = next
			continue
		}
		drainHTTPBody(resp.Body)
		finalHost := hostFromURL(current)
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return chain, "", finalHost, nil
		}
		return chain, "non_success_status", finalHost, nil
	}
	return chain, "redirect_limit_exceeded", hostFromURL(current), nil
}

func resolveCampaignSmokeRedirect(current, location string) (string, error) {
	base, err := url.Parse(current)
	if err != nil {
		return "", err
	}
	next, err := url.Parse(location)
	if err != nil {
		return "", err
	}
	return base.ResolveReference(next).String(), nil
}

func hostFromURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	return u.Host
}

func drainHTTPBody(body io.ReadCloser) {
	if body == nil {
		return
	}
	_, _ = io.Copy(io.Discard, body)
	_ = body.Close()
}
