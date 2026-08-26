package postback

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const microsoftAdsOfflineDefaultURL = "https://campaign.api.bingads.microsoft.com/CampaignManagement/v13/OfflineConversions/Apply"

type MicrosoftAdsAdapter struct{}

type microsoftAdsOfflineConversion struct {
	ConversionCurrencyCode string  `json:"ConversionCurrencyCode,omitempty"`
	ConversionName         string  `json:"ConversionName"`
	ConversionTime         string  `json:"ConversionTime"`
	ConversionValue        float64 `json:"ConversionValue,omitempty"`
	MicrosoftClickID       string  `json:"MicrosoftClickId"`
	HashedEmailAddress     string  `json:"HashedEmailAddress,omitempty"`
	HashedPhoneNumber      string  `json:"HashedPhoneNumber,omitempty"`
}

type microsoftAdsOfflinePayload struct {
	OfflineConversions []microsoftAdsOfflineConversion `json:"OfflineConversions"`
}

type microsoftAdsConfig struct {
	AccountID      string
	CustomerID     string
	ConversionName string
}

func parseMicrosoftAdsConfig(urlTemplate string) (microsoftAdsConfig, error) {
	t := strings.TrimSpace(urlTemplate)
	if t == "" {
		return microsoftAdsConfig{}, fmt.Errorf("microsoft_ads: url_template required (account_id|customer_id|conversion_name)")
	}
	if strings.HasPrefix(t, "http") {
		return microsoftAdsConfig{ConversionName: "conversion"}, nil
	}
	parts := strings.Split(t, "|")
	if len(parts) != 3 {
		return microsoftAdsConfig{}, fmt.Errorf("microsoft_ads: url_template must be account_id|customer_id|conversion_name")
	}
	cfg := microsoftAdsConfig{
		AccountID:      strings.TrimSpace(parts[0]),
		CustomerID:     strings.TrimSpace(parts[1]),
		ConversionName: strings.TrimSpace(parts[2]),
	}
	if cfg.AccountID == "" || cfg.CustomerID == "" || cfg.ConversionName == "" {
		return microsoftAdsConfig{}, fmt.Errorf("microsoft_ads: account_id, customer_id, and conversion_name are required")
	}
	return cfg, nil
}

func parseMicrosoftAccessToken(apiTokenDecrypted, developerTokenFromConfig string) (accessToken, developerToken string) {
	raw := strings.TrimSpace(apiTokenDecrypted)
	if raw == "" {
		return "", strings.TrimSpace(developerTokenFromConfig)
	}
	if strings.HasPrefix(raw, "{") {
		var parsed struct {
			AccessToken    string `json:"access_token"`
			DeveloperToken string `json:"developer_token"`
		}
		if json.Unmarshal([]byte(raw), &parsed) == nil {
			access := strings.TrimSpace(parsed.AccessToken)
			dev := strings.TrimSpace(parsed.DeveloperToken)
			if dev == "" {
				dev = strings.TrimSpace(developerTokenFromConfig)
			}
			return access, dev
		}
	}
	return raw, strings.TrimSpace(developerTokenFromConfig)
}

func (a *MicrosoftAdsAdapter) Send(ctx context.Context, client *http.Client, payload *PostbackPayload, urlTemplate, apiTokenDecrypted string) error {
	clickID := resolveMicrosoftClickID(payload)
	if clickID == "" {
		return fmt.Errorf("microsoft_ads: missing msclkid on conversion payload")
	}

	cfg, err := parseMicrosoftAdsConfig(urlTemplate)
	if err != nil {
		return err
	}

	accessToken, developerToken := parseMicrosoftAccessToken(apiTokenDecrypted, payload.TestEventCode)
	if accessToken == "" {
		return fmt.Errorf("microsoft_ads: oauth access token required")
	}
	if developerToken == "" && !strings.HasPrefix(strings.TrimSpace(urlTemplate), "http") {
		return fmt.Errorf("microsoft_ads: developer token required (test_event_code or api_token JSON)")
	}

	endpoint := microsoftAdsOfflineDefaultURL
	if t := strings.TrimSpace(urlTemplate); strings.HasPrefix(t, "http") {
		endpoint = t
	}

	conv := microsoftAdsOfflineConversion{
		ConversionName:   cfg.ConversionName,
		ConversionTime:   time.Now().UTC().Format("2006-01-02T15:04:05"),
		MicrosoftClickID: clickID,
	}
	if payload.PayoutMicro > 0 {
		conv.ConversionValue = payload.PayoutDollarsAPI()
		conv.ConversionCurrencyCode = "USD"
	}
	if payload.Email != "" {
		conv.HashedEmailAddress = hashSHA256(payload.Email)
	}
	if payload.Phone != "" {
		conv.HashedPhoneNumber = hashSHA256(payload.Phone)
	}

	bodyBytes, err := json.Marshal(microsoftAdsOfflinePayload{
		OfflineConversions: []microsoftAdsOfflineConversion{conv},
	})
	if err != nil {
		return fmt.Errorf("microsoft_ads: marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("microsoft_ads: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("AuthenticationToken", accessToken)
	if developerToken != "" {
		req.Header.Set("DeveloperToken", developerToken)
	}
	if cfg.CustomerID != "" {
		req.Header.Set("CustomerId", cfg.CustomerID)
	}
	if cfg.AccountID != "" {
		req.Header.Set("CustomerAccountId", cfg.AccountID)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("microsoft_ads: http request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	return checkHTTPResponse(resp)
}
