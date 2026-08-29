package provider

import (
	"bytes"
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
	"ad-event-processor/pkg/money"
)

func fetchPropellerAdsCosts(ctx context.Context, client *http.Client, baseURL string, cred Credential, date time.Time) ([]CostLine, error) {
	base := baseURL
	if base == "" {
		base = "https://ssp-api.propellerads.com/v5"
	}
	if client == nil {
		client = &http.Client{Timeout: 60 * time.Second}
	}

	token := cred.APIKey
	if token == "" {
		token = cred.AccessToken
	}
	if token == "" {
		return nil, fmt.Errorf("propellerads: missing api token")
	}

	dateStr := date.UTC().Format("2006-01-02")
	if !database.ValidGAQLDate(dateStr) {
		return nil, fmt.Errorf("propellerads: invalid date %q", dateStr)
	}

	reqBody, err := json.Marshal(map[string]any{
		"group_by":  []string{"campaign_id"},
		"day_from":  dateStr,
		"day_to":    dateStr,
		"date_from": dateStr,
		"date_to":   dateStr,
	})
	if err != nil {
		return nil, err
	}

	endpoint := strings.TrimRight(base, "/") + "/adv/statistics"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
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
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, fmt.Errorf("propellerads report: read body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("propellerads report: status %d: %s", resp.StatusCode, string(body))
	}

	rows, err := parsePropellerAdsStatRows(body)
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
			CampaignID:  uuid.NewSHA1(cred.CustomerID, []byte("propellerads:"+row.campaignID)),
			Date:        date,
			Network:     "propellerads",
			PlacementID: row.campaignID,
			LineType:    LineTypeSpend,
			AmountMicro: row.spendMicro,
			Currency:    "USD",
		})
	}
	return lines, nil
}

type propellerStatRow struct {
	campaignID string
	spendMicro int64
}

func parsePropellerAdsStatRows(body []byte) ([]propellerStatRow, error) {
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
		return mapPropellerAdsItems(items)
	}

	var items []map[string]any
	if err := json.Unmarshal(body, &items); err == nil && len(items) > 0 {
		return mapPropellerAdsItems(items)
	}
	return nil, nil
}

func mapPropellerAdsItems(items []map[string]any) ([]propellerStatRow, error) {
	out := make([]propellerStatRow, 0, len(items))
	for _, item := range items {
		campKey := propellerCampaignKey(item)
		if campKey == "" {
			continue
		}
		spendMicro, err := propellerSpendMicro(item)
		if err != nil || spendMicro == 0 {
			continue
		}
		out = append(out, propellerStatRow{campaignID: campKey, spendMicro: spendMicro})
	}
	return out, nil
}

func propellerCampaignKey(item map[string]any) string {
	for _, key := range []string{"campaign_id", "campaignId", "campaign"} {
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

func propellerSpendMicro(item map[string]any) (int64, error) {
	for _, key := range []string{"spent", "money", "spend", "cost"} {
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
