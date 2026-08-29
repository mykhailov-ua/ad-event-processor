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

func fetchPopAdsCosts(ctx context.Context, client *http.Client, baseURL string, cred Credential, date time.Time) ([]CostLine, error) {
	base := baseURL
	if base == "" {
		base = "https://www.popads.net"
	}
	if client == nil {
		client = &http.Client{Timeout: 60 * time.Second}
	}

	apiKey := cred.APIKey
	if apiKey == "" {
		apiKey = cred.AccessToken
	}
	if apiKey == "" {
		return nil, fmt.Errorf("popads: missing api key")
	}

	dateStr := date.UTC().Format("2006-01-02")
	if !database.ValidGAQLDate(dateStr) {
		return nil, fmt.Errorf("popads: invalid date %q", dateStr)
	}

	form := url.Values{}
	form.Set("key", apiKey)
	form.Set("zone", "UTC")
	form.Set("groups", "campaign")
	form.Set("start", dateStr+" 00:00")
	form.Set("end", dateStr+" 23:59")

	endpoint := strings.TrimRight(base, "/") + "/api/report_advertiser"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

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
		return nil, fmt.Errorf("popads report: read body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("popads report: status %d: %s", resp.StatusCode, string(body))
	}

	rows, err := parseNetworkStatRows(body, []string{"campaign_id", "campaignId", "campaignID", "campaign"})
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
			CampaignID:  uuid.NewSHA1(cred.CustomerID, []byte("popads:"+row.campaignID)),
			Date:        date,
			Network:     "popads",
			PlacementID: row.campaignID,
			LineType:    LineTypeSpend,
			AmountMicro: row.spendMicro,
			Currency:    "USD",
		})
	}
	return lines, nil
}
