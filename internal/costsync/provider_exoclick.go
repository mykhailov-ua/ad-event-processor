package costsync

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

type exoClickLoginResponse struct {
	Token     string `json:"token"`
	Type      string `json:"type"`
	ExpiresIn int    `json:"expires_in"`
}

type exoClickGlobalStatsResponse struct {
	Result []struct {
		Cost    float64 `json:"cost"`
		GroupBy struct {
			CampaignID struct {
				ID string `json:"id"`
			} `json:"campaign_id"`
		} `json:"group_by"`
	} `json:"result"`
}

func fetchExoClickCosts(ctx context.Context, client *http.Client, baseURL string, cred Credential, date time.Time) ([]CostLine, error) {
	base := baseURL
	if base == "" {
		base = "https://api.exoclick.com/v2"
	}
	if client == nil {
		client = &http.Client{Timeout: 60 * time.Second}
	}

	authType, token, err := exoClickSessionToken(ctx, client, base, cred)
	if err != nil {
		return nil, err
	}

	dateStr := date.UTC().Format("2006-01-02")
	if !database.ValidGAQLDate(dateStr) {
		return nil, fmt.Errorf("exoclick: invalid date %q", dateStr)
	}

	reqBody, err := json.Marshal(map[string]any{
		"group_by": []string{"campaign_id"},
		"filter": map[string]string{
			"date_from": dateStr,
			"date_to":   dateStr,
		},
		"limit":  500,
		"offset": 0,
	})
	if err != nil {
		return nil, err
	}

	endpoint := strings.TrimRight(base, "/") + "/statistics/a/global"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", strings.TrimSpace(authType)+" "+strings.TrimSpace(token))

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
		return nil, fmt.Errorf("exoclick report: read body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("exoclick report: status %d: %s", resp.StatusCode, string(body))
	}

	var parsed exoClickGlobalStatsResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, err
	}

	lines := make([]CostLine, 0, len(parsed.Result))
	for _, row := range parsed.Result {
		spendMicro, err := money.JSONAmountToMicro(row.Cost)
		if err != nil || spendMicro == 0 {
			continue
		}
		campKey := strings.TrimSpace(row.GroupBy.CampaignID.ID)
		if campKey == "" {
			continue
		}
		lines = append(lines, CostLine{
			CustomerID:  cred.CustomerID,
			CampaignID:  uuid.NewSHA1(cred.CustomerID, []byte("exoclick:"+campKey)),
			Date:        date,
			Network:     "exoclick",
			PlacementID: campKey,
			LineType:    LineTypeSpend,
			AmountMicro: spendMicro,
			Currency:    "USD",
		})
	}
	return lines, nil
}

func exoClickSessionToken(ctx context.Context, client *http.Client, base string, cred Credential) (authType string, token string, err error) {
	if cred.AccessToken != "" {
		authType = "Bearer"
		if cred.ExtraConfig != nil {
			if t := strings.TrimSpace(cred.ExtraConfig["auth_type"]); t != "" {
				authType = t
			}
		}
		return authType, cred.AccessToken, nil
	}

	apiToken := cred.APIKey
	if apiToken == "" {
		apiToken = cred.ExtraConfig["api_token"]
	}
	if apiToken == "" {
		return "", "", fmt.Errorf("exoclick: missing api token")
	}

	loginBody, err := json.Marshal(map[string]string{"api_token": apiToken})
	if err != nil {
		return "", "", err
	}
	endpoint := strings.TrimRight(base, "/") + "/login"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(loginBody))
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		coldpath.CloseHTTPResponse(resp)
		return "", "", err
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", "", fmt.Errorf("exoclick login: read body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("exoclick login: status %d: %s", resp.StatusCode, string(body))
	}

	var parsed exoClickLoginResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", "", err
	}
	if strings.TrimSpace(parsed.Token) == "" {
		return "", "", fmt.Errorf("exoclick login: empty session token")
	}
	authType = strings.TrimSpace(parsed.Type)
	if authType == "" {
		authType = "Bearer"
	}
	return authType, parsed.Token, nil
}
