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
)

func fetchRichAdsCosts(ctx context.Context, client *http.Client, baseURL string, cred Credential, date time.Time) ([]CostLine, error) {
	base := baseURL
	if base == "" {
		base = "https://api.richads.com"
	}
	if client == nil {
		client = &http.Client{Timeout: 60 * time.Second}
	}

	apiKey := strings.TrimSpace(cred.APIKey)
	if apiKey == "" {
		apiKey = strings.TrimSpace(cred.AccessToken)
	}
	if apiKey == "" {
		return nil, fmt.Errorf("richads: missing api key")
	}

	dateStr := date.UTC().Format("2006-01-02")
	if !database.ValidGAQLDate(dateStr) {
		return nil, fmt.Errorf("richads: invalid date %q", dateStr)
	}

	segment := strings.TrimSpace(cred.ExtraConfig["segment"])
	if segment == "" {
		segment = "campaign_id"
	}

	q := url.Values{}
	q.Set("from", dateStr)
	q.Set("to", dateStr)
	q.Set("segment", segment)
	q.Set("output", "json")
	q.Set("api_key", apiKey)

	endpoint := fmt.Sprintf("%s/api/reports/?%s", strings.TrimRight(base, "/"), q.Encode())
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
		return nil, fmt.Errorf("richads report: read body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("richads report: status %d: %s", resp.StatusCode, string(body))
	}

	rows, err := parseRichAdsStatRows(body)
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
			CampaignID:  uuid.NewSHA1(cred.CustomerID, []byte("richads:"+row.campaignID)),
			Date:        date,
			Network:     "richads",
			PlacementID: row.campaignID,
			LineType:    LineTypeSpend,
			AmountMicro: row.spendMicro,
			Currency:    "USD",
		})
	}
	return lines, nil
}

func parseRichAdsStatRows(body []byte) ([]networkStatRow, error) {
	var top map[string]json.RawMessage
	if err := json.Unmarshal(body, &top); err != nil {
		return nil, err
	}

	raw, ok := top["response"]
	if ok && len(raw) > 0 {
		var wrapped struct {
			Result []map[string]any `json:"result"`
		}
		if err := json.Unmarshal(raw, &wrapped); err == nil && len(wrapped.Result) > 0 {
			return mapNetworkStatItems(wrapped.Result, []string{"campaign_id", "campaignId", "campaign", "id"})
		}
	}

	return parseNetworkStatRows(body, []string{"campaign_id", "campaignId", "campaign", "id"})
}
