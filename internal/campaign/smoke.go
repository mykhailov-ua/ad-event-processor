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
