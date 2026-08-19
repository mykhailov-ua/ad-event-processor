package costsync

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/bidshard/ad-event-processor/pkg/coldpath"
	"github.com/bidshard/ad-event-processor/pkg/money"
)

type taboolaReportResponse struct {
	Results []struct {
		CampaignID   int64   `json:"campaign"`
		CampaignName string  `json:"campaign_name"`
		Placement    string  `json:"site"`
		Spent        float64 `json:"spent"`
		Currency     string  `json:"currency"`
	} `json:"results"`
}

func fetchTaboolaCosts(ctx context.Context, client *http.Client, baseURL string, cred Credential, date time.Time) ([]CostLine, error) {
	base := baseURL
	if base == "" {
		base = "https://backstage.taboola.com/backstage/api/1.0"
	}
	if client == nil {
		client = &http.Client{Timeout: 60 * time.Second}
	}

	accountID := cred.AccountID
	if accountID == "" {
		accountID = cred.ExtraConfig["account_id"]
	}
	if accountID == "" {
		return nil, fmt.Errorf("taboola: missing account id")
	}

	endpoint := fmt.Sprintf("%s/%s/reports/campaign-summary/dimensions/campaign_site_day_breakdown?start_date=%s&end_date=%s",
		strings.TrimRight(base, "/"), accountID, date.Format("2006-01-02"), date.Format("2006-01-02"))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, http.NoBody)
	if err != nil {
		return nil, err
	}
	if cred.AccessToken != "" {
		req.Header.Set("Authorization", "Bearer "+cred.AccessToken)
	} else if cred.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+cred.APIKey)
	}

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
		return nil, fmt.Errorf("taboola report: read body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("taboola report: status %d: %s", resp.StatusCode, string(body))
	}

	var parsed taboolaReportResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, err
	}

	lines := make([]CostLine, 0, len(parsed.Results))
	for _, row := range parsed.Results {
		spendMicro, err := money.JSONAmountToMicro(row.Spent)
		if err != nil || spendMicro == 0 {
			continue
		}
		campKey := fmt.Sprintf("%d", row.CampaignID)
		lines = append(lines, CostLine{
			CustomerID:  cred.CustomerID,
			CampaignID:  uuid.NewSHA1(cred.CustomerID, []byte("taboola:"+campKey)),
			Date:        date,
			Network:     "taboola",
			PlacementID: row.Placement,
			LineType:    LineTypeSpend,
			AmountMicro: spendMicro,
			Currency:    row.Currency,
		})
	}
	return lines, nil
}

type outbrainReportResponse struct {
	CampaignResults []struct {
		Metadata struct {
			ID int64 `json:"id"`
		} `json:"metadata"`
		Metrics struct {
			Spend float64 `json:"spend"`
		} `json:"metrics"`
		Sections []struct {
			Metadata struct {
				ID int64 `json:"id"`
			} `json:"metadata"`
			Metrics struct {
				Spend float64 `json:"spend"`
			} `json:"metrics"`
		} `json:"sections"`
	} `json:"campaignResults"`
}

func fetchOutbrainCosts(ctx context.Context, client *http.Client, baseURL string, cred Credential, date time.Time) ([]CostLine, error) {
	base := baseURL
	if base == "" {
		base = "https://api.outbrain.com/amplify/v0.1"
	}
	if client == nil {
		client = &http.Client{Timeout: 60 * time.Second}
	}

	endpoint := fmt.Sprintf("%s/reports/marketers/%s/campaigns?from=%s&to=%s&breakdown=section",
		strings.TrimRight(base, "/"), cred.AccountID, date.Format("2006-01-02"), date.Format("2006-01-02"))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, http.NoBody)
	if err != nil {
		return nil, err
	}
	if cred.AccessToken != "" {
		req.Header.Set("Authorization", "Bearer "+cred.AccessToken)
	}

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
		return nil, fmt.Errorf("taboola report: read body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("outbrain report: status %d: %s", resp.StatusCode, string(body))
	}

	var parsed outbrainReportResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, err
	}

	lines := make([]CostLine, 0)
	for _, camp := range parsed.CampaignResults {
		campKey := fmt.Sprintf("%d", camp.Metadata.ID)
		for _, sec := range camp.Sections {
			spendMicro, err := money.JSONAmountToMicro(sec.Metrics.Spend)
			if err != nil || spendMicro <= 0 {
				continue
			}
			lines = append(lines, CostLine{
				CustomerID:  cred.CustomerID,
				CampaignID:  uuid.NewSHA1(cred.CustomerID, []byte("outbrain:"+campKey)),
				Date:        date,
				Network:     "outbrain",
				PlacementID: fmt.Sprintf("%d", sec.Metadata.ID),
				LineType:    LineTypeSpend,
				AmountMicro: spendMicro,
				Currency:    "USD",
			})
		}
	}
	return lines, nil
}
