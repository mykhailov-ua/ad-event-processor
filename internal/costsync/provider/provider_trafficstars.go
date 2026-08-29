package provider

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
	"ad-event-processor/pkg/money"
)

type trafficStarsStatisticsResponse struct {
	Data []struct {
		ID    int64   `json:"id"`
		Costs float64 `json:"costs"`
	} `json:"data"`
}

func fetchTrafficStarsCosts(ctx context.Context, client *http.Client, baseURL string, cred Credential, date time.Time) ([]CostLine, error) {
	base := baseURL
	if base == "" {
		base = "https://api.trafficstars.com"
	}
	if client == nil {
		client = &http.Client{Timeout: 60 * time.Second}
	}

	token, err := trafficStarsAccessToken(ctx, client, base, cred)
	if err != nil {
		return nil, err
	}

	dateStr := date.UTC().Format("2006-01-02")
	if !database.ValidGAQLDate(dateStr) {
		return nil, fmt.Errorf("trafficstars: invalid date %q", dateStr)
	}

	reqBody, err := json.Marshal(map[string]string{
		"date_from": dateStr,
		"date_to":   dateStr,
		"group_by":  "campaign",
	})
	if err != nil {
		return nil, err
	}

	endpoint := strings.TrimRight(base, "/") + "/v2/campaigns/statistics"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
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
	body, err := readLimitedHTTPBody(resp, 4<<20)
	if err != nil {
		return nil, fmt.Errorf("trafficstars report: read body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("trafficstars report: status %d: %s", resp.StatusCode, string(body))
	}

	var parsed trafficStarsStatisticsResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, err
	}

	lines := make([]CostLine, 0, len(parsed.Data))
	for _, row := range parsed.Data {
		if row.ID == 0 {
			continue
		}
		spendMicro, err := money.JSONAmountToMicro(row.Costs)
		if err != nil || spendMicro == 0 {
			continue
		}
		campKey := fmt.Sprintf("%d", row.ID)
		lines = append(lines, CostLine{
			CustomerID:  cred.CustomerID,
			CampaignID:  uuid.NewSHA1(cred.CustomerID, []byte("trafficstars:"+campKey)),
			Date:        date,
			Network:     "trafficstars",
			PlacementID: campKey,
			LineType:    LineTypeSpend,
			AmountMicro: spendMicro,
			Currency:    "USD",
		})
	}
	return lines, nil
}

func trafficStarsAccessToken(ctx context.Context, client *http.Client, base string, cred Credential) (string, error) {
	token := strings.TrimSpace(cred.AccessToken)
	if token != "" {
		return token, nil
	}
	offlineKey := strings.TrimSpace(cred.RefreshToken)
	if offlineKey == "" {
		offlineKey = strings.TrimSpace(cred.APIKey)
	}
	if offlineKey == "" {
		return "", fmt.Errorf("trafficstars: missing access token or offline api key")
	}
	access, _, err := RefreshTrafficStarsOAuth(ctx, client, base, Credential{RefreshToken: offlineKey})
	if err != nil {
		return "", err
	}
	return access, nil
}
