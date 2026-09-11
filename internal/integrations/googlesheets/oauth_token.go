package googlesheets

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

func exchangeAuthorizationCode(
	ctx context.Context,
	client *http.Client,
	cfg Config,
	code, redirectURI string,
) (accessToken, refreshToken string, expires time.Time, err error) {
	if cfg.ClientID == "" || cfg.ClientSecret == "" {
		return "", "", time.Time{}, ErrNotConfigured
	}
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("client_id", cfg.ClientID)
	form.Set("client_secret", cfg.ClientSecret)
	form.Set("redirect_uri", redirectURI)

	tokenURL := "https://oauth2.googleapis.com/token"
	if cfg.TokenURL != "" {
		tokenURL = cfg.TokenURL
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", "", time.Time{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return parseTokenResponse(ctx, client, req)
}

func refreshAccessToken(
	ctx context.Context,
	client *http.Client,
	cfg Config,
	refreshToken string,
) (accessToken, newRefresh string, expires time.Time, err error) {
	if refreshToken == "" {
		return "", "", time.Time{}, fmt.Errorf("google oauth: missing refresh token")
	}
	if cfg.ClientID == "" || cfg.ClientSecret == "" {
		return "", "", time.Time{}, ErrNotConfigured
	}
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", refreshToken)
	form.Set("client_id", cfg.ClientID)
	form.Set("client_secret", cfg.ClientSecret)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://oauth2.googleapis.com/token", strings.NewReader(form.Encode()))
	if err != nil {
		return "", "", time.Time{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	access, _, expiresAt, err := parseTokenResponse(ctx, client, req)
	return access, refreshToken, expiresAt, err
}

func parseTokenResponse(ctx context.Context, client *http.Client, req *http.Request) (string, string, time.Time, error) {
	resp, err := client.Do(req)
	if err != nil {
		coldpath.CloseHTTPResponse(resp)
		return "", "", time.Time{}, err
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8192))
	if err != nil {
		return "", "", time.Time{}, fmt.Errorf("google oauth: read body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", "", time.Time{}, fmt.Errorf("google oauth: status %d: %s", resp.StatusCode, string(body))
	}
	var parsed struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int64  `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", "", time.Time{}, err
	}
	expires := time.Now().Add(time.Duration(parsed.ExpiresIn) * time.Second)
	return strings.TrimSpace(parsed.AccessToken), strings.TrimSpace(parsed.RefreshToken), expires, nil
}
