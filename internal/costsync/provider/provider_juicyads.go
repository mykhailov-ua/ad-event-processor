package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"ad-event-processor/internal/database"
	"ad-event-processor/pkg/coldpath"
)

const juicyAdsAPIBaseDefault = "https://api.juicyads.com"

func fetchJuicyAdsCosts(ctx context.Context, client *http.Client, baseURL string, cred Credential, date time.Time) ([]CostLine, error) {
	base := baseURL
	if base == "" {
		base = juicyAdsAPIBaseDefault
	}
	if client == nil {
		client = &http.Client{Timeout: 60 * time.Second}
	}

	token := strings.TrimSpace(cred.APIKey)
	if token == "" {
		token = strings.TrimSpace(cred.AccessToken)
	}
	if token == "" {
		return nil, fmt.Errorf("juicyads: missing api token")
	}

	dateStr := date.UTC().Format("2006-01-02")
	if !database.ValidGAQLDate(dateStr) {
		return nil, fmt.Errorf("juicyads: invalid date %q", dateStr)
	}

	campaignType := strings.TrimSpace(cred.ExtraConfig["campaign_type"])
	if campaignType == "" {
		campaignType = "popunders"
	}

	campaigns, err := juicyAdsListCampaigns(ctx, client, base, token, campaignType)
	if err != nil {
		return nil, err
	}

	lines := make([]CostLine, 0, len(campaigns))
	for _, campID := range campaigns {
		spendMicro, err := juicyAdsCampaignSpend(ctx, client, base, token, campaignType, campID, dateStr)
		if err != nil || spendMicro == 0 {
			continue
		}
		lines = append(lines, CostLine{
			CustomerID:  cred.CustomerID,
			CampaignID:  uuid.NewSHA1(cred.CustomerID, []byte("juicyads:"+campID)),
			Date:        date,
			Network:     "juicyads",
			PlacementID: campID,
			LineType:    LineTypeSpend,
			AmountMicro: spendMicro,
			Currency:    "USD",
		})
	}
	return lines, nil
}

func juicyAdsListCampaigns(ctx context.Context, client *http.Client, base, token, campaignType string) ([]string, error) {
	endpoint := fmt.Sprintf("%s/campaigns/%s/%s", strings.TrimRight(base, "/"), campaignType, token)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, http.NoBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		coldpath.CloseHTTPResponse(resp)
		return nil, err
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()
	body, err := readLimitedHTTPBody(resp, 4<<20)
	if err != nil {
		return nil, fmt.Errorf("juicyads campaigns: read body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("juicyads campaigns: status %d: %s", resp.StatusCode, string(body))
	}

	var items []map[string]any
	if err := json.Unmarshal(body, &items); err != nil {
		return nil, err
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		if id := networkCampaignKey(item, []string{"id", "campaign_id", "campaignId"}); id != "" {
			out = append(out, id)
		}
	}
	return out, nil
}

func juicyAdsCampaignSpend(ctx context.Context, client *http.Client, base, token, campaignType, campaignID, dateStr string) (int64, error) {
	endpoint := fmt.Sprintf("%s/statistics/%s/advertiser/%s/%s/%s/%s",
		strings.TrimRight(base, "/"), campaignType, token, campaignID, dateStr, dateStr)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, http.NoBody)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		coldpath.CloseHTTPResponse(resp)
		return 0, err
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()
	body, err := readLimitedHTTPBody(resp, 4<<20)
	if err != nil {
		return 0, fmt.Errorf("juicyads stats: read body: %w", err)
	}
	if resp.StatusCode == http.StatusNotFound {
		return 0, nil
	}
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("juicyads stats: status %d: %s", resp.StatusCode, string(body))
	}

	var rows []map[string]any
	if err := json.Unmarshal(body, &rows); err != nil {
		return 0, err
	}
	var total int64
	for _, row := range rows {
		spendMicro, err := networkSpendMicro(row)
		if err != nil {
			return 0, err
		}
		total += spendMicro
	}
	if total == 0 {
		return 0, nil
	}
	return total, nil
}
