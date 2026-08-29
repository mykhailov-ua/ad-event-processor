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
	"ad-event-processor/pkg/money"
)

func fetchAdsterraCosts(ctx context.Context, client *http.Client, baseURL string, cred Credential, date time.Time) ([]CostLine, error) {
	base := baseURL
	if base == "" {
		base = "https://api3.adsterratools.com"
	}
	if client == nil {
		client = &http.Client{Timeout: 60 * time.Second}
	}

	apiKey := cred.APIKey
	if apiKey == "" {
		apiKey = cred.AccessToken
	}
	if apiKey == "" {
		return nil, fmt.Errorf("adsterra: missing api token")
	}

	dateStr := date.UTC().Format("2006-01-02")
	if !database.ValidGAQLDate(dateStr) {
		return nil, fmt.Errorf("adsterra: invalid date %q", dateStr)
	}

	q := url.Values{}
	q.Set("start_date", dateStr)
	q.Set("finish_date", dateStr)
	q.Set("group_by", "campaign")

	endpoint := fmt.Sprintf("%s/advertiser/stats.json?%s", strings.TrimRight(base, "/"), q.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, http.NoBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-API-Key", apiKey)

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
		return nil, fmt.Errorf("adsterra report: read body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("adsterra report: status %d: %s", resp.StatusCode, string(body))
	}

	rows, err := parseAdsterraStatRows(body)
	if err != nil {
		return nil, err
	}

	lines := make([]CostLine, 0, len(rows))
	for _, row := range rows {
		if row.spendMicro == 0 || row.campaignID == "" {
			continue
		}
		lines = append(lines, CostLine{
			CustomerID:  cred.CustomerID,
			CampaignID:  uuid.NewSHA1(cred.CustomerID, []byte("adsterra:"+row.campaignID)),
			Date:        date,
			Network:     "adsterra",
			PlacementID: row.campaignID,
			LineType:    LineTypeSpend,
			AmountMicro: row.spendMicro,
			Currency:    "USD",
		})
	}
	return lines, nil
}

type adsterraStatRow struct {
	campaignID string
	spendMicro int64
}

func parseAdsterraStatRows(body []byte) ([]adsterraStatRow, error) {
	var top map[string]json.RawMessage
	if err := json.Unmarshal(body, &top); err != nil {
		return nil, err
	}

	for _, key := range []string{"items", "result", "data", "rows"} {
		raw, ok := top[key]
		if !ok || len(raw) == 0 {
			continue
		}
		var items []map[string]any
		if err := json.Unmarshal(raw, &items); err != nil {
			continue
		}
		return mapAdsterraItems(items)
	}

	var items []map[string]any
	if err := json.Unmarshal(body, &items); err == nil && len(items) > 0 {
		return mapAdsterraItems(items)
	}

	out := make([]adsterraStatRow, 0)
	for key, raw := range top {
		if key == "message" || key == "status" {
			continue
		}
		var item map[string]any
		if err := json.Unmarshal(raw, &item); err != nil {
			continue
		}
		row, ok := adsterraRowFromMap(item)
		if !ok {
			row.campaignID = strings.TrimSpace(key)
			var spendErr error
			row.spendMicro, spendErr = adsterraSpendMicro(item)
			ok = spendErr == nil && row.spendMicro > 0 && row.campaignID != ""
		}
		if ok {
			out = append(out, row)
		}
	}
	return out, nil
}

func mapAdsterraItems(items []map[string]any) ([]adsterraStatRow, error) {
	out := make([]adsterraStatRow, 0, len(items))
	for _, item := range items {
		if row, ok := adsterraRowFromMap(item); ok {
			out = append(out, row)
		}
	}
	return out, nil
}

func adsterraRowFromMap(item map[string]any) (adsterraStatRow, bool) {
	campKey := adsterraCampaignKey(item)
	spendMicro, err := adsterraSpendMicro(item)
	if err != nil || spendMicro == 0 || campKey == "" {
		return adsterraStatRow{}, false
	}
	return adsterraStatRow{campaignID: campKey, spendMicro: spendMicro}, true
}

func adsterraCampaignKey(item map[string]any) string {
	for _, key := range []string{"campaign_id", "campaignId", "campaign", "id"} {
		if v, ok := item[key]; ok {
			switch t := v.(type) {
			case string:
				if s := strings.TrimSpace(t); s != "" {
					return s
				}
			case float64:
				return fmt.Sprintf("%.0f", t)
			}
		}
	}
	return ""
}

func adsterraSpendMicro(item map[string]any) (int64, error) {
	for _, key := range []string{"spent", "spend", "cost", "money"} {
		v, ok := item[key]
		if !ok {
			continue
		}
		switch t := v.(type) {
		case string:
			return money.ParseDecimal(strings.TrimSpace(t))
		case float64:
			return money.JSONAmountToMicro(t)
		}
	}
	return 0, nil
}
