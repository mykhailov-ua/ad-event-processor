package costsync

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

func fetchRevcontentCosts(ctx context.Context, client *http.Client, baseURL string, cred Credential, date time.Time) ([]CostLine, error) {
	base := baseURL
	if base == "" {
		base = "https://api.revcontent.io"
	}
	if client == nil {
		client = &http.Client{Timeout: 60 * time.Second}
	}

	token := cred.AccessToken
	if token == "" {
		token = cred.APIKey
	}
	if token == "" {
		return nil, fmt.Errorf("revcontent: missing access token")
	}

	dateStr := date.UTC().Format("2006-01-02")
	if !database.ValidGAQLDate(dateStr) {
		return nil, fmt.Errorf("revcontent: invalid date %q", dateStr)
	}

	q := url.Values{}
	q.Set("date_from", dateStr)
	q.Set("date_to", dateStr)
	q.Set("limit", "1000")
	q.Set("offset", "0")

	endpoint := fmt.Sprintf("%s/stats/api/v1.0/boosts?%s", strings.TrimRight(base, "/"), q.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, http.NoBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
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
		return nil, fmt.Errorf("revcontent report: read body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("revcontent report: status %d: %s", resp.StatusCode, string(body))
	}

	rows, err := parseNetworkStatRows(body, []string{"id", "boost_id", "campaign_id", "campaignId"})
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
			CampaignID:  uuid.NewSHA1(cred.CustomerID, []byte("revcontent:"+row.campaignID)),
			Date:        date,
			Network:     "revcontent",
			PlacementID: row.campaignID,
			LineType:    LineTypeSpend,
			AmountMicro: row.spendMicro,
			Currency:    "USD",
		})
	}
	return lines, nil
}
