package platformsync

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"ad-event-processor/internal/costsync"
	"ad-event-processor/pkg/coldpath"
)

type fbCampaignResponse struct {
	Status      string `json:"status"`
	DailyBudget string `json:"daily_budget"`
}

func fetchFacebookCampaignStatus(ctx context.Context, client *http.Client, baseURL string, cred costsync.Credential, externalCampaignID string) (RemoteCampaignStatus, error) {
	base := baseURL
	if base == "" {
		base = "https://graph.facebook.com/v19.0"
	}
	if client == nil {
		client = &http.Client{Timeout: 60 * time.Second}
	}
	if externalCampaignID == "" {
		return RemoteCampaignStatus{}, fmt.Errorf("facebook: missing external campaign id")
	}

	q := url.Values{}
	q.Set("fields", "status,daily_budget")
	q.Set("access_token", cred.AccessToken)
	endpoint := fmt.Sprintf("%s/%s?%s", strings.TrimRight(base, "/"), externalCampaignID, q.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, http.NoBody)
	if err != nil {
		return RemoteCampaignStatus{}, err
	}

	resp, err := client.Do(req)
	if err != nil {
		coldpath.CloseHTTPResponse(resp)
		return RemoteCampaignStatus{}, err
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return RemoteCampaignStatus{}, err
	}
	if resp.StatusCode != http.StatusOK {
		return RemoteCampaignStatus{}, fmt.Errorf("facebook campaign: status %d: %s", resp.StatusCode, string(body))
	}

	var parsed fbCampaignResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return RemoteCampaignStatus{}, err
	}

	out := RemoteCampaignStatus{Status: strings.ToUpper(strings.TrimSpace(parsed.Status))}
	if parsed.DailyBudget != "" {
		cents, err := strconv.ParseInt(parsed.DailyBudget, 10, 64)
		if err == nil {
			out.DailyBudgetMicro = cents * 10_000
			out.HasDailyBudgetMicro = true
		}
	}
	return out, nil
}

func mutateFacebookCampaign(ctx context.Context, client *http.Client, baseURL string, cred costsync.Credential, externalCampaignID, action string, req MutationRequest) (map[string]string, error) {
	base := baseURL
	if base == "" {
		base = "https://graph.facebook.com/v19.0"
	}
	if client == nil {
		client = &http.Client{Timeout: 60 * time.Second}
	}

	form := url.Values{}
	form.Set("access_token", cred.AccessToken)
	switch action {
	case ActionPause:
		form.Set("status", "PAUSED")
	case ActionResume:
		form.Set("status", "ACTIVE")
	case ActionSetDailyBudget:
		if req.DailyBudgetMicro <= 0 {
			return nil, fmt.Errorf("facebook: daily_budget_micro required")
		}
		cents := req.DailyBudgetMicro / 10_000
		if cents <= 0 {
			return nil, fmt.Errorf("facebook: daily_budget_micro too small")
		}
		form.Set("daily_budget", strconv.FormatInt(cents, 10))
	default:
		return nil, fmt.Errorf("facebook: unsupported action %q", action)
	}

	endpoint := fmt.Sprintf("%s/%s", strings.TrimRight(base, "/"), externalCampaignID)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(httpReq)
	if err != nil {
		coldpath.CloseHTTPResponse(resp)
		return nil, err
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("facebook mutate: status %d: %s", resp.StatusCode, string(body))
	}
	return map[string]string{"vendor_response": string(body)}, nil
}
