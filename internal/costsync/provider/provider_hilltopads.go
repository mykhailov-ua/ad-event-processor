package provider

import (
	"context"
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

func fetchHilltopAdsCosts(ctx context.Context, client *http.Client, baseURL string, cred Credential, date time.Time) ([]CostLine, error) {
	base := baseURL
	if base == "" {
		base = "https://api.hilltopads.com"
	}
	if client == nil {
		client = &http.Client{Timeout: 60 * time.Second}
	}

	apiKey := cred.APIKey
	if apiKey == "" {
		apiKey = cred.AccessToken
	}
	if apiKey == "" {
		return nil, fmt.Errorf("hilltopads: missing api key")
	}

	dateStr := date.UTC().Format("2006-01-02")
	if !database.ValidGAQLDate(dateStr) {
		return nil, fmt.Errorf("hilltopads: invalid date %q", dateStr)
	}

	q := url.Values{}
	q.Set("key", apiKey)
	q.Set("date", dateStr)
	q.Set("date2", dateStr)
	q.Set("group", "campaignID")

	endpoint := fmt.Sprintf("%s/advertiser/listStats?%s", strings.TrimRight(base, "/"), q.Encode())
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
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, fmt.Errorf("hilltopads report: read body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("hilltopads report: status %d: %s", resp.StatusCode, string(body))
	}

	rows, err := parseNetworkStatRows(body, []string{"campaignID", "campaignId", "campaign_id", "campaign"})
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
			CampaignID:  uuid.NewSHA1(cred.CustomerID, []byte("hilltopads:"+row.campaignID)),
			Date:        date,
			Network:     "hilltopads",
			PlacementID: row.campaignID,
			LineType:    LineTypeSpend,
			AmountMicro: row.spendMicro,
			Currency:    "USD",
		})
	}
	return lines, nil
}
