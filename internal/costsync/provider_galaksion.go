package costsync

import (
	"bytes"
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
	"ad-event-processor/pkg/money"
)

type galaksionAuthResponse struct {
	Token string `json:"token"`
}

type galaksionStatisticsResponse struct {
	Rows []map[string]any `json:"rows"`
}

func fetchGalaksionCosts(ctx context.Context, client *http.Client, baseURL string, cred Credential, date time.Time) ([]CostLine, error) {
	base := baseURL
	if base == "" {
		base = "https://ssp2-api.galaksion.com/api"
	}
	if client == nil {
		client = &http.Client{Timeout: 60 * time.Second}
	}

	token, err := galaksionAuthToken(ctx, client, base, cred)
	if err != nil {
		return nil, err
	}

	dateStr := date.UTC().Format("2006-01-02")
	if !database.ValidGAQLDate(dateStr) {
		return nil, fmt.Errorf("galaksion: invalid date %q", dateStr)
	}

	groupBy := strings.TrimSpace(cred.ExtraConfig["group_by"])
	if groupBy == "" {
		groupBy = `["campaign"]`
	}
	orderBy := strings.TrimSpace(cred.ExtraConfig["order_by"])
	if orderBy == "" {
		orderBy = `{"field":"campaign","direction":"asc"}`
	}

	q := url.Values{}
	q.Set("groupBy", groupBy)
	q.Set("orderBy", orderBy)
	q.Set("dateFrom", dateStr+" 00:00:00")
	q.Set("dateTo", dateStr+" 23:59:59")
	q.Set("limit", "5000")

	endpoint := fmt.Sprintf("%s/v1/advertiser/statistics?%s", strings.TrimRight(base, "/"), q.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, http.NoBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Auth-Token", token)
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
		return nil, fmt.Errorf("galaksion report: read body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("galaksion report: status %d: %s", resp.StatusCode, string(body))
	}

	var parsed galaksionStatisticsResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, err
	}

	lines := make([]CostLine, 0, len(parsed.Rows))
	for _, row := range parsed.Rows {
		campKey := galaksionCampaignKey(row)
		spendMicro, err := galaksionSpendMicro(row)
		if err != nil || spendMicro == 0 || campKey == "" {
			continue
		}
		lines = append(lines, CostLine{
			CustomerID:  cred.CustomerID,
			CampaignID:  uuid.NewSHA1(cred.CustomerID, []byte("galaksion:"+campKey)),
			Date:        date,
			Network:     "galaksion",
			PlacementID: campKey,
			LineType:    LineTypeSpend,
			AmountMicro: spendMicro,
			Currency:    "USD",
		})
	}
	return lines, nil
}

func galaksionAuthToken(ctx context.Context, client *http.Client, base string, cred Credential) (string, error) {
	token := strings.TrimSpace(cred.AccessToken)
	if token == "" {
		token = strings.TrimSpace(cred.APIKey)
	}
	if token != "" {
		return token, nil
	}

	email := strings.TrimSpace(cred.AccountID)
	if email == "" {
		email = strings.TrimSpace(cred.ExtraConfig["email"])
	}
	password := strings.TrimSpace(cred.ExtraConfig["password"])
	if password == "" {
		password = strings.TrimSpace(cred.RefreshToken)
	}
	if email == "" || password == "" {
		return "", fmt.Errorf("galaksion: missing api token or login email/password")
	}

	loginBody, err := json.Marshal(map[string]string{
		"email":    email,
		"password": password,
	})
	if err != nil {
		return "", err
	}

	endpoint := strings.TrimRight(base, "/") + "/v1/auth"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(loginBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		coldpath.CloseHTTPResponse(resp)
		return "", err
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()
	body, err := readLimitedHTTPBody(resp, 1<<20)
	if err != nil {
		return "", fmt.Errorf("galaksion login: read body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("galaksion login: status %d: %s", resp.StatusCode, string(body))
	}

	var parsed galaksionAuthResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", err
	}
	if strings.TrimSpace(parsed.Token) == "" {
		return "", fmt.Errorf("galaksion login: empty token")
	}
	return parsed.Token, nil
}

func galaksionCampaignKey(row map[string]any) string {
	for _, key := range []string{"campaign", "campaign_id", "campaignId", "id"} {
		if v, ok := row[key]; ok {
			switch t := v.(type) {
			case string:
				if s := strings.TrimSpace(t); s != "" {
					return s
				}
			case float64:
				return fmt.Sprintf("%.0f", t)
			}
		}
	}
	return ""
}

func galaksionSpendMicro(row map[string]any) (int64, error) {
	v, ok := row["money"]
	if !ok {
		return networkSpendMicro(row)
	}
	switch t := v.(type) {
	case string:
		return money.ParseDecimal(strings.TrimSpace(t))
	case float64:
		return money.JSONAmountToMicro(t)
	default:
		return 0, nil
	}
}
