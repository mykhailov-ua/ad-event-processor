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

const linkedInAPIVersionDefault = "202508"

type linkedInAnalyticsResponse struct {
	Elements []struct {
		CostInUSD   string   `json:"costInUsd"`
		CostInLocal string   `json:"costInLocalCurrency"`
		PivotValues []string `json:"pivotValues"`
		DateRange   struct {
			Start struct {
				Day   int `json:"day"`
				Month int `json:"month"`
				Year  int `json:"year"`
			} `json:"start"`
		} `json:"dateRange"`
	} `json:"elements"`
}

func fetchLinkedInCosts(ctx context.Context, client *http.Client, baseURL string, cred Credential, date time.Time) ([]CostLine, error) {
	base := baseURL
	if base == "" {
		base = "https://api.linkedin.com/rest"
	}
	if client == nil {
		client = &http.Client{Timeout: 60 * time.Second}
	}

	accountID := strings.TrimSpace(cred.AccountID)
	if accountID == "" {
		accountID = strings.TrimSpace(cred.ExtraConfig["ad_account_id"])
	}
	if accountID == "" {
		return nil, fmt.Errorf("linkedin: missing ad account id")
	}
	token := strings.TrimSpace(cred.AccessToken)
	if token == "" {
		token = strings.TrimSpace(cred.APIKey)
	}
	if token == "" {
		return nil, fmt.Errorf("linkedin: missing access token")
	}

	dateStr := date.UTC().Format("2006-01-02")
	if !database.ValidGAQLDate(dateStr) {
		return nil, fmt.Errorf("linkedin: invalid date %q", dateStr)
	}

	apiVersion := strings.TrimSpace(cred.ExtraConfig["linkedin_version"])
	if apiVersion == "" {
		apiVersion = linkedInAPIVersionDefault
	}

	accountURN := linkedInAccountURN(accountID)
	dateRange := fmt.Sprintf("(start:(year:%d,month:%d,day:%d),end:(year:%d,month:%d,day:%d))",
		date.Year(), int(date.Month()), date.Day(),
		date.Year(), int(date.Month()), date.Day())

	q := url.Values{}
	q.Set("q", "analytics")
	q.Set("pivot", "CAMPAIGN")
	q.Set("timeGranularity", "DAILY")
	q.Set("dateRange", dateRange)
	q.Set("accounts", "List("+accountURN+")")
	q.Set("fields", "costInUsd,costInLocalCurrency,dateRange,pivotValues")

	endpoint := strings.TrimRight(base, "/") + "/adAnalytics?" + q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, http.NoBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Linkedin-Version", apiVersion)
	req.Header.Set("X-Restli-Protocol-Version", "2.0.0")

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
		return nil, fmt.Errorf("linkedin analytics: read body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("linkedin analytics: status %d: %s", resp.StatusCode, string(body))
	}

	var parsed linkedInAnalyticsResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, err
	}

	lines := make([]CostLine, 0, len(parsed.Elements))
	for _, row := range parsed.Elements {
		if len(row.PivotValues) == 0 {
			continue
		}
		if row.DateRange.Start.Year != 0 &&
			(row.DateRange.Start.Year != date.Year() || row.DateRange.Start.Month != int(date.Month()) || row.DateRange.Start.Day != date.Day()) {
			continue
		}
		spendRaw := strings.TrimSpace(row.CostInUSD)
		if spendRaw == "" {
			spendRaw = strings.TrimSpace(row.CostInLocal)
		}
		spendMicro, err := money.ParseDecimal(spendRaw)
		if err != nil || spendMicro == 0 {
			continue
		}
		campaignID := linkedInCampaignIDFromPivot(row.PivotValues[0])
		if campaignID == "" {
			continue
		}
		lines = append(lines, CostLine{
			CustomerID:  cred.CustomerID,
			CampaignID:  uuid.NewSHA1(cred.CustomerID, []byte("linkedin:"+campaignID)),
			Date:        date,
			Network:     "linkedin",
			PlacementID: campaignID,
			LineType:    LineTypeSpend,
			AmountMicro: spendMicro,
			Currency:    "USD",
		})
	}
	return lines, nil
}

func linkedInAccountURN(accountID string) string {
	if strings.HasPrefix(accountID, "urn:") {
		return accountID
	}
	return "urn:li:sponsoredAccount:" + accountID
}

func linkedInCampaignIDFromPivot(pivot string) string {
	const prefix = "urn:li:sponsoredCampaign:"
	pivot = strings.TrimSpace(pivot)
	if strings.HasPrefix(pivot, prefix) {
		return strings.TrimPrefix(pivot, prefix)
	}
	if idx := strings.LastIndex(pivot, ":"); idx >= 0 && idx+1 < len(pivot) {
		if _, err := strconv.ParseInt(pivot[idx+1:], 10, 64); err == nil {
			return pivot[idx+1:]
		}
	}
	return pivot
}
