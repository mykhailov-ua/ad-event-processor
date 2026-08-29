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

type tiktokCampaignListResponse struct {
	Data struct {
		List []struct {
			CampaignID      string `json:"campaign_id"`
			OperationStatus string `json:"operation_status"`
			Budget          string `json:"budget"`
			BudgetMode      string `json:"budget_mode"`
		} `json:"list"`
	} `json:"data"`
}

type tiktokStatusUpdateRequest struct {
	AdvertiserID    string   `json:"advertiser_id"`
	CampaignIDs     []string `json:"campaign_ids"`
	OperationStatus string   `json:"operation_status"`
}

func tiktokAdvertiserID(cred costsync.Credential) (string, error) {
	advertiserID := strings.TrimSpace(cred.AccountID)
	if advertiserID == "" {
		advertiserID = strings.TrimSpace(cred.ExtraConfig["advertiser_id"])
	}
	if advertiserID == "" {
		return "", fmt.Errorf("tiktok: missing advertiser id")
	}
	return advertiserID, nil
}

func tiktokAccessToken(cred costsync.Credential) (string, error) {
	token := strings.TrimSpace(cred.AccessToken)
	if token == "" {
		token = strings.TrimSpace(cred.APIKey)
	}
	if token == "" {
		return "", fmt.Errorf("tiktok: missing access token")
	}
	return token, nil
}

func fetchTikTokCampaignStatus(ctx context.Context, client *http.Client, baseURL string, cred costsync.Credential, externalCampaignID string) (RemoteCampaignStatus, error) {
	base := baseURL
	if base == "" {
		base = "https://business-api.tiktok.com/open_api/v1.3"
	}
	if client == nil {
		client = &http.Client{Timeout: 60 * time.Second}
	}
	if externalCampaignID == "" {
		return RemoteCampaignStatus{}, fmt.Errorf("tiktok: missing external campaign id")
	}

	advertiserID, err := tiktokAdvertiserID(cred)
	if err != nil {
		return RemoteCampaignStatus{}, err
	}
	token, err := tiktokAccessToken(cred)
	if err != nil {
		return RemoteCampaignStatus{}, err
	}

	filtering, err := json.Marshal(map[string][]string{
		"campaign_ids": {externalCampaignID},
	})
	if err != nil {
		return RemoteCampaignStatus{}, err
	}

	q := url.Values{}
	q.Set("advertiser_id", advertiserID)
	q.Set("filtering", string(filtering))
	endpoint := fmt.Sprintf("%s/campaign/get/?%s", strings.TrimRight(base, "/"), q.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, http.NoBody)
	if err != nil {
		return RemoteCampaignStatus{}, err
	}
	req.Header.Set("Access-Token", token)

	resp, err := client.Do(req)
	if err != nil {
		coldpath.CloseHTTPResponse(resp)
		return RemoteCampaignStatus{}, err
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return RemoteCampaignStatus{}, err
	}
	if resp.StatusCode != http.StatusOK {
		return RemoteCampaignStatus{}, fmt.Errorf("tiktok campaign: status %d: %s", resp.StatusCode, string(body))
	}

	var parsed tiktokCampaignListResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return RemoteCampaignStatus{}, err
	}
	if len(parsed.Data.List) == 0 {
		return RemoteCampaignStatus{}, fmt.Errorf("tiktok: campaign not found")
	}

	row := parsed.Data.List[0]
	out := RemoteCampaignStatus{Status: tiktokOperationStatusToCanonical(row.OperationStatus)}
	if row.Budget != "" && strings.EqualFold(row.BudgetMode, "BUDGET_MODE_DAY") {
		budget, parseErr := strconv.ParseFloat(strings.TrimSpace(row.Budget), 64)
		if parseErr == nil && budget > 0 {
			out.DailyBudgetMicro = int64(budget * 1_000_000)
			out.HasDailyBudgetMicro = true
		}
	}
	return out, nil
}

func mutateTikTokCampaign(ctx context.Context, client *http.Client, baseURL string, cred costsync.Credential, externalCampaignID, action string, req MutationRequest) (map[string]string, error) {
	base := baseURL
	if base == "" {
		base = "https://business-api.tiktok.com/open_api/v1.3"
	}
	if client == nil {
		client = &http.Client{Timeout: 60 * time.Second}
	}
	if externalCampaignID == "" {
		return nil, fmt.Errorf("tiktok: missing external campaign id")
	}

	advertiserID, err := tiktokAdvertiserID(cred)
	if err != nil {
		return nil, err
	}
	token, err := tiktokAccessToken(cred)
	if err != nil {
		return nil, err
	}

	var operationStatus string
	switch action {
	case ActionPause:
		operationStatus = "DISABLE"
	case ActionResume:
		operationStatus = "ENABLE"
	case ActionSetDailyBudget:
		return nil, fmt.Errorf("tiktok: daily budget mutation not supported")
	default:
		return nil, fmt.Errorf("tiktok: unsupported action %q", action)
	}

	payload, err := json.Marshal(tiktokStatusUpdateRequest{
		AdvertiserID:    advertiserID,
		CampaignIDs:     []string{externalCampaignID},
		OperationStatus: operationStatus,
	})
	if err != nil {
		return nil, err
	}

	endpoint := fmt.Sprintf("%s/campaign/status/update/", strings.TrimRight(base, "/"))
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(string(payload)))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Access-Token", token)

	resp, err := client.Do(httpReq)
	if err != nil {
		coldpath.CloseHTTPResponse(resp)
		return nil, err
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tiktok mutate: status %d: %s", resp.StatusCode, string(body))
	}
	return map[string]string{"vendor_response": string(body)}, nil
}

func tiktokOperationStatusToCanonical(status string) string {
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case "ENABLE":
		return "ACTIVE"
	case "DISABLE":
		return "PAUSED"
	default:
		return strings.ToUpper(strings.TrimSpace(status))
	}
}
