package platformsync

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"ad-event-processor/internal/costsync"
	"ad-event-processor/pkg/coldpath"
)

const microsoftAdsCampaignBaseDefault = "https://campaign.api.bingads.microsoft.com/CampaignManagement/v13"

type microsoftAdsCampaignsResponse struct {
	Campaigns []struct {
		ID     int64  `json:"Id"`
		Status string `json:"Status"`
	} `json:"Campaigns"`
}

func microsoftAdsCredentialFields(cred costsync.Credential) (accountID, customerID, developerToken string, err error) {
	accountID = strings.TrimSpace(cred.AccountID)
	if accountID == "" {
		accountID = strings.TrimSpace(cred.ExtraConfig["customer_account_id"])
	}
	customerID = strings.TrimSpace(cred.ExtraConfig["customer_id"])
	developerToken = strings.TrimSpace(cred.ExtraConfig["developer_token"])
	if accountID == "" {
		return "", "", "", fmt.Errorf("microsoft_ads: missing account id")
	}
	if customerID == "" {
		return "", "", "", fmt.Errorf("microsoft_ads: missing customer_id in extra_config")
	}
	if developerToken == "" {
		return "", "", "", fmt.Errorf("microsoft_ads: missing developer_token in extra_config")
	}
	return accountID, customerID, developerToken, nil
}

func microsoftAdsSetHeaders(req *http.Request, token, customerID, accountID, developerToken string) {
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("CustomerId", customerID)
	req.Header.Set("CustomerAccountId", accountID)
	req.Header.Set("DeveloperToken", developerToken)
}

func fetchMicrosoftAdsCampaignStatus(ctx context.Context, client *http.Client, baseURL string, cred costsync.Credential, externalCampaignID string) (RemoteCampaignStatus, error) {
	base := baseURL
	if base == "" {
		base = microsoftAdsCampaignBaseDefault
	}
	if client == nil {
		client = &http.Client{Timeout: 60 * time.Second}
	}
	if externalCampaignID == "" {
		return RemoteCampaignStatus{}, fmt.Errorf("microsoft_ads: missing external campaign id")
	}

	campaignID, err := strconv.ParseInt(strings.TrimSpace(externalCampaignID), 10, 64)
	if err != nil {
		return RemoteCampaignStatus{}, fmt.Errorf("microsoft_ads: invalid external campaign id")
	}

	accountID, customerID, developerToken, err := microsoftAdsCredentialFields(cred)
	if err != nil {
		return RemoteCampaignStatus{}, err
	}
	token := strings.TrimSpace(cred.AccessToken)
	if token == "" {
		return RemoteCampaignStatus{}, fmt.Errorf("microsoft_ads: missing access token")
	}

	accountNum, err := strconv.ParseInt(accountID, 10, 64)
	if err != nil {
		return RemoteCampaignStatus{}, fmt.Errorf("microsoft_ads: account id must be numeric: %w", err)
	}

	payload, err := json.Marshal(map[string]any{
		"AccountId":   accountNum,
		"CampaignIds": []int64{campaignID},
	})
	if err != nil {
		return RemoteCampaignStatus{}, err
	}

	endpoint := strings.TrimRight(base, "/") + "/Campaigns/GetCampaignsByIds"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return RemoteCampaignStatus{}, err
	}
	microsoftAdsSetHeaders(req, token, customerID, accountID, developerToken)
	req.Header.Set("Content-Type", "application/json")

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
		return RemoteCampaignStatus{}, fmt.Errorf("microsoft_ads campaign: status %d: %s", resp.StatusCode, string(body))
	}

	var parsed microsoftAdsCampaignsResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return RemoteCampaignStatus{}, err
	}
	if len(parsed.Campaigns) == 0 {
		return RemoteCampaignStatus{}, fmt.Errorf("microsoft_ads: campaign not found")
	}

	return RemoteCampaignStatus{Status: microsoftAdsStatusToCanonical(parsed.Campaigns[0].Status)}, nil
}

func mutateMicrosoftAdsCampaign(ctx context.Context, client *http.Client, baseURL string, cred costsync.Credential, externalCampaignID, action string, req MutationRequest) (map[string]string, error) {
	base := baseURL
	if base == "" {
		base = microsoftAdsCampaignBaseDefault
	}
	if client == nil {
		client = &http.Client{Timeout: 60 * time.Second}
	}
	if externalCampaignID == "" {
		return nil, fmt.Errorf("microsoft_ads: missing external campaign id")
	}

	campaignID, err := strconv.ParseInt(strings.TrimSpace(externalCampaignID), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("microsoft_ads: invalid external campaign id")
	}

	accountID, customerID, developerToken, err := microsoftAdsCredentialFields(cred)
	if err != nil {
		return nil, err
	}
	token := strings.TrimSpace(cred.AccessToken)
	if token == "" {
		return nil, fmt.Errorf("microsoft_ads: missing access token")
	}

	accountNum, err := strconv.ParseInt(accountID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("microsoft_ads: account id must be numeric: %w", err)
	}

	var vendorStatus string
	switch action {
	case ActionPause:
		vendorStatus = "Paused"
	case ActionResume:
		vendorStatus = "Active"
	case ActionSetDailyBudget:
		return nil, fmt.Errorf("microsoft_ads: daily budget mutation not supported")
	default:
		return nil, fmt.Errorf("microsoft_ads: unsupported action %q", action)
	}

	payload, err := json.Marshal(map[string]any{
		"AccountId": accountNum,
		"Campaigns": []map[string]any{{
			"Id":     campaignID,
			"Status": vendorStatus,
		}},
	})
	if err != nil {
		return nil, err
	}

	endpoint := strings.TrimRight(base, "/") + "/Campaigns/UpdateCampaigns"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	microsoftAdsSetHeaders(httpReq, token, customerID, accountID, developerToken)
	httpReq.Header.Set("Content-Type", "application/json")

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
		return nil, fmt.Errorf("microsoft_ads mutate: status %d: %s", resp.StatusCode, string(body))
	}
	return map[string]string{"vendor_response": string(body)}, nil
}

func microsoftAdsStatusToCanonical(status string) string {
	switch strings.TrimSpace(status) {
	case "Active":
		return "ACTIVE"
	case "Paused":
		return "PAUSED"
	default:
		return strings.ToUpper(strings.TrimSpace(status))
	}
}
