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
	"ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
)

const (
	campaignSmokeMaxRedirects = 8
	campaignSmokeTimeout      = 15 * time.Second
)

var errCampaignSmokeTrackerBaseMissing = errors.New("campaign smoke tracker base missing")

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

func (h *CampaignsHTTPHandlers) registerCampaignSmokeRoutes(mux *http.ServeMux, limit func(http.HandlerFunc) http.HandlerFunc, perm func([]string, http.HandlerFunc) http.HandlerFunc) {
	write := []string{"campaigns:write"}
	mux.HandleFunc("POST /api/v1/campaigns/{id}/smoke", limit(perm(write, h.postCampaignSmoke)))
}

type campaignSmokeRunner interface {
	RunCampaignSmoke(ctx context.Context, campaignID uuid.UUID) (CampaignSmokeResultDTO, error)
}

func (h *CampaignsHTTPHandlers) postCampaignSmoke(w http.ResponseWriter, r *http.Request) {
	campaignID, ok := h.ParseCampaignID(w, r)
	if !ok {
		return
	}
	runner, ok := h.Campaigns.(campaignSmokeRunner)
	if !ok {
		httpresponse.Error(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "smoke check not configured")
		return
	}
	result, err := runner.RunCampaignSmoke(r.Context(), campaignID)
	if err != nil {
		if h.WriteServiceError != nil {
			h.WriteServiceError(w, err)
			return
		}
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL", err.Error())
		return
	}
	httpresponse.JSON(w, http.StatusOK, result)
}

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
