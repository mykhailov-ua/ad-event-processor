package platformsync

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"ad-event-processor/internal/costsync"
	"ad-event-processor/internal/database"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/pkg/coldpath"
)

type googleCampaignQueryResponse struct {
	Results []struct {
		Campaign struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"campaign"`
		CampaignBudget struct {
			ResourceName string `json:"resourceName"`
			AmountMicros string `json:"amountMicros"`
		} `json:"campaignBudget"`
	} `json:"results"`
}

func fetchGoogleCampaignStatus(ctx context.Context, client *http.Client, baseURL string, cred costsync.Credential, externalCampaignID string) (RemoteCampaignStatus, error) {
	base := baseURL
	if base == "" {
		base = "https://googleads.googleapis.com/v16"
	}
	if client == nil {
		client = &http.Client{Timeout: 60 * time.Second}
	}
	if externalCampaignID == "" {
		return RemoteCampaignStatus{}, fmt.Errorf("google ads: missing external campaign id")
	}
	if !database.ValidGAQLDigits(externalCampaignID) {
		return RemoteCampaignStatus{}, fmt.Errorf("google ads: invalid external campaign id")
	}

	query := "SELECT campaign.id, campaign.status, campaign_budget.resource_name, campaign_budget.amount_micros FROM campaign WHERE campaign.id = " + externalCampaignID
	payload, err := json.Marshal(map[string]string{"query": query})
	if err != nil {
		return RemoteCampaignStatus{}, err
	}

	customerID := cred.AccountID
	if customerID == "" {
		customerID = cred.ExtraConfig["customer_id"]
	}
	if customerID == "" {
		return RemoteCampaignStatus{}, fmt.Errorf("google ads: missing customer id")
	}

	endpoint := fmt.Sprintf("%s/customers/%s/googleAds:searchStream", strings.TrimRight(base, "/"), customerID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(string(payload)))
	if err != nil {
		return RemoteCampaignStatus{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cred.AccessToken)
	if devToken := cred.ExtraConfig["developer_token"]; devToken != "" {
		req.Header.Set("developer-token", devToken)
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
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return RemoteCampaignStatus{}, err
	}
	if resp.StatusCode != http.StatusOK {
		return RemoteCampaignStatus{}, fmt.Errorf("google ads campaign: status %d: %s", resp.StatusCode, string(body))
	}

	var parsed googleCampaignQueryResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return RemoteCampaignStatus{}, err
	}
	if len(parsed.Results) == 0 {
		return RemoteCampaignStatus{}, fmt.Errorf("google ads: campaign not found")
	}
	row := parsed.Results[0]
	out := RemoteCampaignStatus{
		Status:         strings.ToUpper(strings.TrimSpace(row.Campaign.Status)),
		BudgetResource: row.CampaignBudget.ResourceName,
	}
	if row.CampaignBudget.AmountMicros != "" {
		micro, parseErr := strconv.ParseInt(row.CampaignBudget.AmountMicros, 10, 64)
		if parseErr == nil {
			out.DailyBudgetMicro = micro
			out.HasDailyBudgetMicro = true
		}
	}
	return out, nil
}

func mutateGoogleCampaign(ctx context.Context, client *http.Client, baseURL string, cred costsync.Credential, link db.PlatformCampaignLink, action string, req MutationRequest) (map[string]string, error) {
	base := baseURL
	if base == "" {
		base = "https://googleads.googleapis.com/v16"
	}
	if client == nil {
		client = &http.Client{Timeout: 60 * time.Second}
	}

	customerID := link.AccountID
	if customerID == "" {
		customerID = cred.AccountID
	}
	if customerID == "" {
		customerID = cred.ExtraConfig["customer_id"]
	}
	if customerID == "" {
		return nil, fmt.Errorf("google ads: missing customer id")
	}

	var operations []map[string]any
	switch action {
	case ActionPause:
		operations = []map[string]any{{
			"updateMask": "status",
			"update": map[string]string{
				"resourceName": fmt.Sprintf("customers/%s/campaigns/%s", customerID, link.ExternalCampaignID),
				"status":       "PAUSED",
			},
		}}
	case ActionResume:
		operations = []map[string]any{{
			"updateMask": "status",
			"update": map[string]string{
				"resourceName": fmt.Sprintf("customers/%s/campaigns/%s", customerID, link.ExternalCampaignID),
				"status":       "ENABLED",
			},
		}}
	case ActionSetDailyBudget:
		if req.DailyBudgetMicro <= 0 {
			return nil, fmt.Errorf("google ads: daily_budget_micro required")
		}
		budgetResource := link.ExternalBudgetResource
		if budgetResource == "" {
			remote, err := fetchGoogleCampaignStatus(ctx, client, base, cred, link.ExternalCampaignID)
			if err != nil {
				return nil, err
			}
			budgetResource = remote.BudgetResource
		}
		if budgetResource == "" {
			return nil, fmt.Errorf("google ads: missing campaign budget resource")
		}
		operations = []map[string]any{{
			"updateMask": "amountMicros",
			"update": map[string]string{
				"resourceName": budgetResource,
				"amountMicros": strconv.FormatInt(req.DailyBudgetMicro, 10),
			},
		}}
	default:
		return nil, fmt.Errorf("google ads: unsupported action %q", action)
	}

	payload, err := json.Marshal(map[string]any{"operations": operations})
	if err != nil {
		return nil, err
	}

	endpoint := fmt.Sprintf("%s/customers/%s/googleAds:mutate", strings.TrimRight(base, "/"), customerID)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(string(payload)))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+cred.AccessToken)
	if devToken := cred.ExtraConfig["developer_token"]; devToken != "" {
		httpReq.Header.Set("developer-token", devToken)
	}

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
		return nil, fmt.Errorf("google ads mutate: status %d: %s", resp.StatusCode, string(body))
	}
	return map[string]string{"vendor_response": string(body)}, nil
}
