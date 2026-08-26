package costsync

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
)

const evadavAPIBaseDefault = "https://evadavapi.com/api/v2.2"

func fetchEvadavCosts(ctx context.Context, client *http.Client, baseURL string, cred Credential, date time.Time) ([]CostLine, error) {
	base := baseURL
	if base == "" {
		base = evadavAPIBaseDefault
	}
	if client == nil {
		client = &http.Client{Timeout: 60 * time.Second}
	}

	apiKey := strings.TrimSpace(cred.APIKey)
	if apiKey == "" {
		apiKey = strings.TrimSpace(cred.AccessToken)
	}
	if apiKey == "" {
		return nil, fmt.Errorf("evadav: missing api key")
	}

	dateStr := date.UTC().Format("2006-01-02")
	if !database.ValidGAQLDate(dateStr) {
		return nil, fmt.Errorf("evadav: invalid date %q", dateStr)
	}
	day := date.UTC().Format("02.01.2006")

	payload, err := json.Marshal(map[string]string{"day": day})
	if err != nil {
		return nil, err
	}

	endpoint := strings.TrimRight(base, "/") + "/advertiser/stats/campaign"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Api-Key", apiKey)

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
		return nil, fmt.Errorf("evadav report: read body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("evadav report: status %d: %s", resp.StatusCode, string(body))
	}

	rows, err := parseEvadavCampaignStatRows(body)
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
			CampaignID:  uuid.NewSHA1(cred.CustomerID, []byte("evadav:"+row.campaignID)),
			Date:        date,
			Network:     "evadav",
			PlacementID: row.campaignID,
			LineType:    LineTypeSpend,
			AmountMicro: row.spendMicro,
			Currency:    "USD",
		})
	}
	return lines, nil
}

func parseEvadavCampaignStatRows(body []byte) ([]networkStatRow, error) {
	var parsed struct {
		Success bool `json:"success"`
		Data    struct {
			Stat []map[string]any `json:"stat"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, err
	}
	if !parsed.Success && len(parsed.Data.Stat) == 0 {
		return nil, nil
	}
	return mapNetworkStatItems(parsed.Data.Stat, []string{"campaignId", "campaign_id", "campaign"})
}
