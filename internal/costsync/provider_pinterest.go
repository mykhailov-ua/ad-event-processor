package costsync

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"

	"ad-event-processor/internal/database"
	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/money"
)

const pinterestCampaignAnalyticsBatch = 100

type pinterestCampaignsListResponse struct {
	Items []struct {
		ID string `json:"id"`
	} `json:"items"`
	Bookmark string `json:"bookmark"`
}

func fetchPinterestCosts(ctx context.Context, client *http.Client, baseURL string, cred Credential, date time.Time) ([]CostLine, error) {
	base := baseURL
	if base == "" {
		base = "https://api.pinterest.com/v5"
	}
	if client == nil {
		client = &http.Client{Timeout: 90 * time.Second}
	}

	adAccountID := strings.TrimSpace(cred.AccountID)
	if adAccountID == "" {
		adAccountID = strings.TrimSpace(cred.ExtraConfig["ad_account_id"])
	}
	if adAccountID == "" {
		return nil, fmt.Errorf("pinterest: missing ad account id")
	}
	token := strings.TrimSpace(cred.AccessToken)
	if token == "" {
		token = strings.TrimSpace(cred.APIKey)
	}
	if token == "" {
		return nil, fmt.Errorf("pinterest: missing access token")
	}

	dateStr := date.UTC().Format("2006-01-02")
	if !database.ValidGAQLDate(dateStr) {
		return nil, fmt.Errorf("pinterest: invalid date %q", dateStr)
	}

	campaignIDs, err := pinterestListCampaignIDs(ctx, client, base, adAccountID, token)
	if err != nil {
		return nil, err
	}
	if len(campaignIDs) == 0 {
		return nil, nil
	}

	lines := make([]CostLine, 0, len(campaignIDs))
	for start := 0; start < len(campaignIDs); start += pinterestCampaignAnalyticsBatch {
		end := start + pinterestCampaignAnalyticsBatch
		if end > len(campaignIDs) {
			end = len(campaignIDs)
		}
		batchLines, err := pinterestFetchCampaignAnalytics(ctx, client, base, adAccountID, token, dateStr, campaignIDs[start:end], date, cred)
		if err != nil {
			return nil, err
		}
		lines = append(lines, batchLines...)
	}
	return lines, nil
}

func pinterestListCampaignIDs(ctx context.Context, client *http.Client, base, adAccountID, token string) ([]string, error) {
	ids := make([]string, 0, 64)
	bookmark := ""
	for page := 0; page < 50; page++ {
		q := url.Values{}
		q.Set("page_size", "250")
		if bookmark != "" {
			q.Set("bookmark", bookmark)
		}
		endpoint := fmt.Sprintf("%s/ad_accounts/%s/campaigns?%s", strings.TrimRight(base, "/"), url.PathEscape(adAccountID), q.Encode())
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, http.NoBody)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := client.Do(req)
		if err != nil {
			coldpath.CloseHTTPResponse(resp)
			return nil, err
		}
		body, err := readLimitedHTTPBody(resp, 4<<20)
		if err != nil {
			return nil, fmt.Errorf("pinterest campaigns: %w", err)
		}
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("pinterest campaigns: status %d: %s", resp.StatusCode, string(body))
		}

		var parsed pinterestCampaignsListResponse
		if err := json.Unmarshal(body, &parsed); err != nil {
			return nil, err
		}
		for _, item := range parsed.Items {
			id := strings.TrimSpace(item.ID)
			if id != "" {
				ids = append(ids, id)
			}
		}
		bookmark = strings.TrimSpace(parsed.Bookmark)
		if bookmark == "" {
			break
		}
	}
	return ids, nil
}

func pinterestFetchCampaignAnalytics(ctx context.Context, client *http.Client, base, adAccountID, token, dateStr string, campaignIDs []string, date time.Time, cred Credential) ([]CostLine, error) {
	q := url.Values{}
	q.Set("start_date", dateStr)
	q.Set("end_date", dateStr)
	q.Set("campaign_ids", strings.Join(campaignIDs, ","))
	q.Set("columns", "SPEND_IN_DOLLAR")
	q.Set("granularity", "DAY")

	endpoint := fmt.Sprintf("%s/ad_accounts/%s/campaigns/analytics?%s", strings.TrimRight(base, "/"), url.PathEscape(adAccountID), q.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, http.NoBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		coldpath.CloseHTTPResponse(resp)
		return nil, err
	}
	body, err := readLimitedHTTPBody(resp, 4<<20)
	if err != nil {
		return nil, fmt.Errorf("pinterest analytics: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("pinterest analytics: status %d: %s", resp.StatusCode, string(body))
	}

	var rows []map[string]json.RawMessage
	if err := json.Unmarshal(body, &rows); err != nil {
		return nil, err
	}

	lines := make([]CostLine, 0, len(rows))
	for _, row := range rows {
		campaignID := pinterestJSONStringField(row, "CAMPAIGN_ID")
		if campaignID == "" {
			continue
		}
		spendMicro, err := pinterestSpendMicro(row)
		if err != nil || spendMicro == 0 {
			continue
		}
		lines = append(lines, CostLine{
			CustomerID:  cred.CustomerID,
			CampaignID:  uuid.NewSHA1(cred.CustomerID, []byte("pinterest:"+campaignID)),
			Date:        date,
			Network:     "pinterest",
			PlacementID: campaignID,
			LineType:    LineTypeSpend,
			AmountMicro: spendMicro,
			Currency:    "USD",
		})
	}
	return lines, nil
}

func pinterestJSONStringField(row map[string]json.RawMessage, key string) string {
	raw, ok := row[key]
	if !ok {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return strings.TrimSpace(s)
	}
	var n json.Number
	if err := json.Unmarshal(raw, &n); err == nil {
		return strings.TrimSpace(n.String())
	}
	return ""
}

func pinterestSpendMicro(row map[string]json.RawMessage) (int64, error) {
	raw, ok := row["SPEND_IN_DOLLAR"]
	if !ok {
		return 0, nil
	}
	var spendFloat float64
	if err := json.Unmarshal(raw, &spendFloat); err == nil {
		return money.JSONAmountToMicro(spendFloat)
	}
	var spendStr string
	if err := json.Unmarshal(raw, &spendStr); err == nil {
		return money.ParseDecimal(spendStr)
	}
	return 0, fmt.Errorf("pinterest: invalid SPEND_IN_DOLLAR")
}
