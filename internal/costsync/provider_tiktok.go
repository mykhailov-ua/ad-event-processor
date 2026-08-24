package costsync

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"

	"ad-event-processor/internal/database"
	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/money"
)

type tiktokReportResponse struct {
	Data struct {
		List []struct {
			Dimensions struct {
				CampaignID  string `json:"campaign_id"`
				StatTimeDay string `json:"stat_time_day"`
			} `json:"dimensions"`
			Metrics struct {
				Spend string `json:"spend"`
			} `json:"metrics"`
		} `json:"list"`
	} `json:"data"`
}

func fetchTikTokCosts(ctx context.Context, client *http.Client, baseURL string, cred Credential, date time.Time) ([]CostLine, error) {
	base := baseURL
	if base == "" {
		base = "https://business-api.tiktok.com/open_api/v1.3"
	}
	if client == nil {
		client = &http.Client{Timeout: 60 * time.Second}
	}

	advertiserID := cred.AccountID
	if advertiserID == "" {
		advertiserID = cred.ExtraConfig["advertiser_id"]
	}
	if advertiserID == "" {
		return nil, fmt.Errorf("tiktok: missing advertiser id")
	}
	token := cred.AccessToken
	if token == "" {
		token = cred.APIKey
	}
	if token == "" {
		return nil, fmt.Errorf("tiktok: missing access token")
	}

	dateStr := date.UTC().Format("2006-01-02")
	if !database.ValidGAQLDate(dateStr) {
		return nil, fmt.Errorf("tiktok: invalid date %q", dateStr)
	}

	q := url.Values{}
	q.Set("advertiser_id", advertiserID)
	q.Set("report_type", "BASIC")
	q.Set("data_level", "AUCTION_CAMPAIGN")
	q.Set("dimensions", `["campaign_id","stat_time_day"]`)
	q.Set("metrics", `["spend"]`)
	q.Set("start_date", dateStr)
	q.Set("end_date", dateStr)
	q.Set("page", "1")
	q.Set("page_size", "1000")

	endpoint := fmt.Sprintf("%s/report/integrated/get/?%s", strings.TrimRight(base, "/"), q.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, http.NoBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Access-Token", token)

	resp, err := client.Do(req)
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
		return nil, fmt.Errorf("tiktok report: read body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tiktok report: status %d: %s", resp.StatusCode, string(body))
	}

	var parsed tiktokReportResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, err
	}

	lines := make([]CostLine, 0, len(parsed.Data.List))
	for _, row := range parsed.Data.List {
		spendMicro, err := money.ParseDecimal(row.Metrics.Spend)
		if err != nil || spendMicro == 0 {
			continue
		}
		campKey := strings.TrimSpace(row.Dimensions.CampaignID)
		if campKey == "" {
			continue
		}
		lines = append(lines, CostLine{
			CustomerID:  cred.CustomerID,
			CampaignID:  uuid.NewSHA1(cred.CustomerID, []byte("tiktok:"+campKey)),
			Date:        date,
			Network:     "tiktok",
			PlacementID: campKey,
			LineType:    LineTypeSpend,
			AmountMicro: spendMicro,
			Currency:    "USD",
		})
	}
	return lines, nil
}
