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

const mondiadAPIBaseDefault = "https://api.members.mondiad.com"

func fetchMondiadCosts(ctx context.Context, client *http.Client, baseURL string, cred Credential, date time.Time) ([]CostLine, error) {
	base := baseURL
	if base == "" {
		base = mondiadAPIBaseDefault
	}
	if client == nil {
		client = &http.Client{Timeout: 60 * time.Second}
	}

	token, err := mondiadBearerToken(ctx, client, base, cred)
	if err != nil {
		return nil, err
	}

	dateStr := date.UTC().Format("2006-01-02")
	if !database.ValidGAQLDate(dateStr) {
		return nil, fmt.Errorf("mondiad: invalid date %q", dateStr)
	}

	q := url.Values{}
	q.Set("startDate", dateStr)
	q.Set("endDate", dateStr)
	q.Set("breakdown", "CAMPAIGN")
	q.Set("size", "500")

	endpoint := fmt.Sprintf("%s/api/1.0/report/advertising/campaign?%s", strings.TrimRight(base, "/"), q.Encode())
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
	body, err := readLimitedHTTPBody(resp, 4<<20)
	if err != nil {
		return nil, fmt.Errorf("mondiad report: read body: %w", err)
	}
	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("mondiad report: status %d: %s", resp.StatusCode, string(body))
	}

	rows, err := parseNetworkStatRows(body, []string{"campaignId", "campaign_id", "campaign"})
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
			CampaignID:  uuid.NewSHA1(cred.CustomerID, []byte("mondiad:"+row.campaignID)),
			Date:        date,
			Network:     "mondiad",
			PlacementID: row.campaignID,
			LineType:    LineTypeSpend,
			AmountMicro: row.spendMicro,
			Currency:    "USD",
		})
	}
	return lines, nil
}

func mondiadBearerToken(ctx context.Context, client *http.Client, base string, cred Credential) (string, error) {
	if token := strings.TrimSpace(cred.AccessToken); token != "" {
		return token, nil
	}
	token, _, _, err := MondiadLogin(ctx, client, base, cred)
	return token, err
}
