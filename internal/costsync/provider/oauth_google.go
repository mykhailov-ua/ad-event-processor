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

func RefreshGoogleOAuth(ctx context.Context, client *http.Client, clientID, clientSecret string, cred Credential) (string, time.Time, error) {
	if cred.RefreshToken == "" {
		return "", time.Time{}, fmt.Errorf("google oauth: missing refresh token")
	}
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}

	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", cred.RefreshToken)
	form.Set("client_id", clientID)
	form.Set("client_secret", clientSecret)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://oauth2.googleapis.com/token", strings.NewReader(form.Encode()))
	if err != nil {
		return "", time.Time{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

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
		return "", time.Time{}, fmt.Errorf("google oauth refresh: status %d: %s", resp.StatusCode, string(body))
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

func RefreshMicrosoftOAuth(ctx context.Context, client *http.Client, clientID, clientSecret string, cred Credential) (string, time.Time, error) {
	if cred.RefreshToken == "" {
		return "", time.Time{}, fmt.Errorf("microsoft_ads oauth: missing refresh token")
	}
	if clientID == "" || clientSecret == "" {
		return "", time.Time{}, fmt.Errorf("microsoft_ads oauth: missing client id or secret")
	}
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}

	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", cred.RefreshToken)
	form.Set("client_id", clientID)
	form.Set("client_secret", clientSecret)
	form.Set("scope", "https://ads.microsoft.com/msads.manage offline_access")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://login.microsoftonline.com/common/oauth2/v2.0/token", strings.NewReader(form.Encode()))
	if err != nil {
		return "", time.Time{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

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
		return "", time.Time{}, fmt.Errorf("microsoft_ads oauth refresh: read body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", time.Time{}, fmt.Errorf("microsoft_ads oauth refresh: status %d: %s", resp.StatusCode, string(body))
	}

	var parsed struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int64  `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", time.Time{}, err
	}
	if parsed.AccessToken == "" {
		return "", time.Time{}, fmt.Errorf("microsoft_ads oauth refresh: empty access_token")
	}
	expiresIn := parsed.ExpiresIn
	if expiresIn <= 0 {
		expiresIn = 3600
	}
	expires := time.Now().Add(time.Duration(expiresIn) * time.Second)
	return parsed.AccessToken, expires, nil
}
