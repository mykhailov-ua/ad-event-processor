package provider

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
)

type snapchatStatsResponse struct {
	RequestStatus   string `json:"request_status"`
	TimeseriesStats []struct {
		SubRequestStatus string `json:"sub_request_status"`
		TimeseriesStat   struct {
			ID          string `json:"id"`
			Type        string `json:"type"`
			Granularity string `json:"granularity"`
			Timeseries  []struct {
				StartTime string `json:"start_time"`
				Stats     struct {
					Spend int64 `json:"spend"`
				} `json:"stats"`
			} `json:"timeseries"`
		} `json:"timeseries_stat"`
	} `json:"timeseries_stats"`
}

func fetchSnapchatCosts(ctx context.Context, client *http.Client, baseURL string, cred Credential, date time.Time) ([]CostLine, error) {
	base := baseURL
	if base == "" {
		base = "https://adsapi.snapchat.com/v1"
	}
	if client == nil {
		client = &http.Client{Timeout: 60 * time.Second}
	}

	adAccountID := strings.TrimSpace(cred.AccountID)
	if adAccountID == "" {
		adAccountID = strings.TrimSpace(cred.ExtraConfig["ad_account_id"])
	}
	if adAccountID == "" {
		return nil, fmt.Errorf("snapchat: missing ad account id")
	}
	token := strings.TrimSpace(cred.AccessToken)
	if token == "" {
		token = strings.TrimSpace(cred.APIKey)
	}
	if token == "" {
		return nil, fmt.Errorf("snapchat: missing access token")
	}

	dateStr := date.UTC().Format("2006-01-02")
	if !database.ValidGAQLDate(dateStr) {
		return nil, fmt.Errorf("snapchat: invalid date %q", dateStr)
	}
	startTime := date.UTC().Format("2006-01-02T15:04:05")
	endTime := date.UTC().Add(24 * time.Hour).Format("2006-01-02T15:04:05")

	q := url.Values{}
	q.Set("breakdown", "campaign")
	q.Set("granularity", "DAY")
	q.Set("fields", "spend")
	q.Set("start_time", startTime)
	q.Set("end_time", endTime)

	endpoint := fmt.Sprintf("%s/adaccounts/%s/stats?%s", strings.TrimRight(base, "/"), url.PathEscape(adAccountID), q.Encode())
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
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, fmt.Errorf("snapchat stats: read body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("snapchat stats: status %d: %s", resp.StatusCode, string(body))
	}

	var parsed snapchatStatsResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, err
	}
	if parsed.RequestStatus != "" && parsed.RequestStatus != "success" && parsed.RequestStatus != "SUCCESS" {
		return nil, fmt.Errorf("snapchat stats: request_status %q", parsed.RequestStatus)
	}

	lines := make([]CostLine, 0, len(parsed.TimeseriesStats))
	for _, block := range parsed.TimeseriesStats {
		if block.SubRequestStatus != "" && block.SubRequestStatus != "success" && block.SubRequestStatus != "SUCCESS" {
			continue
		}
		campaignID := strings.TrimSpace(block.TimeseriesStat.ID)
		if campaignID == "" {
			continue
		}
		var spendMicro int64
		for _, point := range block.TimeseriesStat.Timeseries {
			spendMicro += point.Stats.Spend
		}
		if spendMicro == 0 {
			continue
		}
		lines = append(lines, CostLine{
			CustomerID:  cred.CustomerID,
			CampaignID:  uuid.NewSHA1(cred.CustomerID, []byte("snapchat:"+campaignID)),
			Date:        date,
			Network:     "snapchat",
			PlacementID: campaignID,
			LineType:    LineTypeSpend,
			AmountMicro: spendMicro,
			Currency:    "USD",
		})
	}
	return lines, nil
}
