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

	"github.com/bidshard/ad-event-processor/internal/database"
	"github.com/bidshard/ad-event-processor/pkg/money"
)

type fbInsightsResponse struct {
	Data []struct {
		CampaignID  string `json:"campaign_id"`
		AdsetID     string `json:"adset_id"`
		AdID        string `json:"ad_id"`
		Spend       string `json:"spend"`
		DateStart   string `json:"date_start"`
		Impressions string `json:"impressions"`
	} `json:"data"`
}

func fetchFacebookCosts(ctx context.Context, client *http.Client, baseURL string, cred Credential, date time.Time) ([]CostLine, error) {
	base := baseURL
	if base == "" {
		base = "https://graph.facebook.com/v19.0"
	}
	if client == nil {
		client = &http.Client{Timeout: 60 * time.Second}
	}

	accountID := cred.AccountID
	if accountID == "" {
		accountID = cred.ExtraConfig["ad_account_id"]
	}
	if accountID == "" {
		return nil, fmt.Errorf("facebook: missing ad account id")
	}
	if !strings.HasPrefix(accountID, "act_") {
		accountID = "act_" + accountID
	}

	q := url.Values{}
	q.Set("fields", "campaign_id,adset_id,ad_id,spend,date_start")
	dateStr := date.UTC().Format("2006-01-02")
	if !database.ValidGAQLDate(dateStr) {
		return nil, fmt.Errorf("facebook: invalid date %q", dateStr)
	}
	q.Set("time_range", `{"since":"`+dateStr+`","until":"`+dateStr+`"}`)
	q.Set("level", "ad")
	q.Set("limit", "500")
	q.Set("access_token", cred.AccessToken)

	endpoint := fmt.Sprintf("%s/%s/insights?%s", strings.TrimRight(base, "/"), accountID, q.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("facebook insights: status %d: %s", resp.StatusCode, string(body))
	}

	var parsed fbInsightsResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, err
	}

	lines := make([]CostLine, 0, len(parsed.Data))
	for _, row := range parsed.Data {
		spendMicro, err := money.ParseDecimal(row.Spend)
		if err != nil || spendMicro == 0 {
			continue
		}
		campID, err := uuid.Parse(row.CampaignID)
		if err != nil {
			campID = mapExternalCampaignID(cred.CustomerID, row.CampaignID)
		}
		lines = append(lines, CostLine{
			CustomerID:  cred.CustomerID,
			CampaignID:  campID,
			Date:        date,
			Network:     "facebook",
			PlacementID: row.AdID,
			AdsetID:     row.AdsetID,
			AdID:        row.AdID,
			LineType:    LineTypeSpend,
			AmountMicro: spendMicro,
			Currency:    "USD",
		})
	}
	return lines, nil
}

func mapExternalCampaignID(customerID uuid.UUID, externalID string) uuid.UUID {
	return uuid.NewSHA1(customerID, []byte("fb:"+externalID))
}
