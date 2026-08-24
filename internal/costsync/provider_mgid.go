package costsync

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"ad-event-processor/internal/database"
	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/money"
)

type mgidCampaignStat struct {
	CampaignID int64   `json:"campaign_id"`
	Spent      float64 `json:"spent"`
}

func fetchMGIDCosts(ctx context.Context, client *http.Client, baseURL string, cred Credential, date time.Time) ([]CostLine, error) {
	base := baseURL
	if base == "" {
		base = "https://api.mgid.com/v1"
	}
	if client == nil {
		client = &http.Client{Timeout: 60 * time.Second}
	}

	clientID := cred.AccountID
	if clientID == "" {
		clientID = cred.ExtraConfig["client_id"]
	}
	if clientID == "" {
		return nil, fmt.Errorf("mgid: missing client id")
	}
	token := cred.AccessToken
	if token == "" {
		token = cred.APIKey
	}
	if token == "" {
		return nil, fmt.Errorf("mgid: missing api token")
	}

	dateStr := date.UTC().Format("2006-01-02")
	if !database.ValidGAQLDate(dateStr) {
		return nil, fmt.Errorf("mgid: invalid date %q", dateStr)
	}

	q := url.Values{}
	q.Set("dateInterval", "interval")
	q.Set("startDate", dateStr)
	q.Set("endDate", dateStr)

	endpoint := fmt.Sprintf("%s/goodhits/clients/%s/campaigns-stat?%s",
		strings.TrimRight(base, "/"), url.PathEscape(clientID), q.Encode())
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
		return nil, fmt.Errorf("mgid report: read body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("mgid report: status %d: %s", resp.StatusCode, string(body))
	}

	stats, err := parseMGIDCampaignStats(body)
	if err != nil {
		return nil, err
	}

	lines := make([]CostLine, 0, len(stats))
	for _, row := range stats {
		spendMicro, err := money.JSONAmountToMicro(row.Spent)
		if err != nil || spendMicro == 0 {
			continue
		}
		campKey := strconv.FormatInt(row.CampaignID, 10)
		if campKey == "0" || campKey == "" {
			continue
		}
		lines = append(lines, CostLine{
			CustomerID:  cred.CustomerID,
			CampaignID:  uuid.NewSHA1(cred.CustomerID, []byte("mgid:"+campKey)),
			Date:        date,
			Network:     "mgid",
			PlacementID: campKey,
			LineType:    LineTypeSpend,
			AmountMicro: spendMicro,
			Currency:    "USD",
		})
	}
	return lines, nil
}

func parseMGIDCampaignStats(body []byte) ([]mgidCampaignStat, error) {
	var byKey map[string]mgidCampaignStat
	if err := json.Unmarshal(body, &byKey); err != nil {
		return nil, err
	}
	out := make([]mgidCampaignStat, 0, len(byKey))
	for key, row := range byKey {
		if row.CampaignID == 0 {
			if id, err := strconv.ParseInt(strings.TrimSpace(key), 10, 64); err == nil {
				row.CampaignID = id
			}
		}
		if row.CampaignID == 0 {
			continue
		}
		out = append(out, row)
	}
	return out, nil
}
