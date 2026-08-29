package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"ad-event-processor/pkg/coldpath"
)

func RefreshTikTokOAuth(ctx context.Context, client *http.Client, baseURL, appID, appSecret string, cred Credential) (string, string, time.Time, error) {
	if cred.RefreshToken == "" {
		return "", "", time.Time{}, fmt.Errorf("tiktok oauth: missing refresh token")
	}
	if appID == "" || appSecret == "" {
		return "", "", time.Time{}, fmt.Errorf("tiktok oauth: missing app id or secret")
	}
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	base := baseURL
	if base == "" {
		base = "https://business-api.tiktok.com/open_api/v1.3"
	}

	payload, err := json.Marshal(map[string]string{
		"app_id":        appID,
		"secret":        appSecret,
		"grant_type":    "refresh_token",
		"refresh_token": cred.RefreshToken,
	})
	if err != nil {
		return "", "", time.Time{}, err
	}

	endpoint := strings.TrimRight(base, "/") + "/oauth2/refresh_token/"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(string(payload)))
	if err != nil {
		return "", "", time.Time{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		coldpath.CloseHTTPResponse(resp)
		return "", "", time.Time{}, err
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return "", "", time.Time{}, fmt.Errorf("tiktok oauth refresh: read body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", "", time.Time{}, fmt.Errorf("tiktok oauth refresh: status %d: %s", resp.StatusCode, string(body))
	}

	var parsed struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
			ExpiresIn    int64  `json:"expires_in"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", "", time.Time{}, err
	}
	if parsed.Code != 0 {
		return "", "", time.Time{}, fmt.Errorf("tiktok oauth refresh: api code %d: %s", parsed.Code, parsed.Message)
	}
	if parsed.Data.AccessToken == "" {
		return "", "", time.Time{}, fmt.Errorf("tiktok oauth refresh: empty access_token")
	}
	expiresIn := parsed.Data.ExpiresIn
	if expiresIn <= 0 {
		expiresIn = 24 * 3600
	}
	expires := time.Now().Add(time.Duration(expiresIn) * time.Second)
	return parsed.Data.AccessToken, parsed.Data.RefreshToken, expires, nil
}

func RefreshMetaOAuth(ctx context.Context, client *http.Client, appID, appSecret string, cred Credential) (string, time.Time, error) {
	if cred.RefreshToken == "" {
		return "", time.Time{}, fmt.Errorf("meta oauth: missing refresh token")
	}
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}

	q := url.Values{}
	q.Set("grant_type", "fb_exchange_token")
	q.Set("client_id", appID)
	q.Set("client_secret", appSecret)
	q.Set("fb_exchange_token", cred.RefreshToken)

	endpoint := "https://graph.facebook.com/v19.0/oauth/access_token?" + q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, http.NoBody)
	if err != nil {
		return "", time.Time{}, err
	}
	resp, err := client.Do(req)
	if err != nil {
		coldpath.CloseHTTPResponse(resp)
		return "", time.Time{}, err
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return "", time.Time{}, fmt.Errorf("meta oauth refresh: read body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", time.Time{}, fmt.Errorf("meta oauth refresh: status %d: %s", resp.StatusCode, string(body))
	}

	var parsed struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int64  `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", time.Time{}, err
	}
	expires := time.Now().Add(time.Duration(parsed.ExpiresIn) * time.Second)
	return parsed.AccessToken, expires, nil
}
