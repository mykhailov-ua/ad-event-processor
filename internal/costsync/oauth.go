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

	"ad-event-processor/pkg/coldpath"
)

func refreshTikTokOAuth(ctx context.Context, client *http.Client, baseURL, appID, appSecret string, cred Credential) (string, string, time.Time, error) {
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

func refreshMetaOAuth(ctx context.Context, client *http.Client, appID, appSecret string, cred Credential) (string, time.Time, error) {
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

func refreshGoogleOAuth(ctx context.Context, client *http.Client, clientID, clientSecret string, cred Credential) (string, time.Time, error) {
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

func refreshMicrosoftOAuth(ctx context.Context, client *http.Client, clientID, clientSecret string, cred Credential) (string, time.Time, error) {
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

func refreshSnapchatOAuth(ctx context.Context, client *http.Client, tokenURL, clientID, clientSecret string, cred Credential) (string, string, time.Time, error) {
	if cred.RefreshToken == "" {
		return "", "", time.Time{}, fmt.Errorf("snapchat oauth: missing refresh token")
	}
	if clientID == "" || clientSecret == "" {
		return "", "", time.Time{}, fmt.Errorf("snapchat oauth: missing client id or secret")
	}
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	endpoint := tokenURL
	if endpoint == "" {
		endpoint = "https://accounts.snapchat.com/login/oauth2/access_token"
	}

	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", cred.RefreshToken)
	form.Set("client_id", clientID)
	form.Set("client_secret", clientSecret)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return "", "", time.Time{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

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
		return "", "", time.Time{}, fmt.Errorf("snapchat oauth refresh: read body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", "", time.Time{}, fmt.Errorf("snapchat oauth refresh: status %d: %s", resp.StatusCode, string(body))
	}

	var parsed struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int64  `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", "", time.Time{}, err
	}
	if parsed.AccessToken == "" {
		return "", "", time.Time{}, fmt.Errorf("snapchat oauth refresh: empty access_token")
	}
	expiresIn := parsed.ExpiresIn
	if expiresIn <= 0 {
		expiresIn = 3600
	}
	expires := time.Now().Add(time.Duration(expiresIn) * time.Second)
	refresh := parsed.RefreshToken
	if refresh == "" {
		refresh = cred.RefreshToken
	}
	return parsed.AccessToken, refresh, expires, nil
}

func refreshLinkedInOAuth(ctx context.Context, client *http.Client, tokenURL, clientID, clientSecret string, cred Credential) (string, string, time.Time, error) {
	if cred.RefreshToken == "" {
		return "", "", time.Time{}, fmt.Errorf("linkedin oauth: missing refresh token")
	}
	if clientID == "" || clientSecret == "" {
		return "", "", time.Time{}, fmt.Errorf("linkedin oauth: missing client id or secret")
	}
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	endpoint := tokenURL
	if endpoint == "" {
		endpoint = "https://www.linkedin.com/oauth/v2/accessToken"
	}

	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", cred.RefreshToken)
	form.Set("client_id", clientID)
	form.Set("client_secret", clientSecret)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return "", "", time.Time{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

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
		return "", "", time.Time{}, fmt.Errorf("linkedin oauth refresh: read body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", "", time.Time{}, fmt.Errorf("linkedin oauth refresh: status %d: %s", resp.StatusCode, string(body))
	}

	var parsed struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int64  `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", "", time.Time{}, err
	}
	if parsed.AccessToken == "" {
		return "", "", time.Time{}, fmt.Errorf("linkedin oauth refresh: empty access_token")
	}
	expiresIn := parsed.ExpiresIn
	if expiresIn <= 0 {
		expiresIn = 1800
	}
	expires := time.Now().Add(time.Duration(expiresIn) * time.Second)
	refresh := parsed.RefreshToken
	if refresh == "" {
		refresh = cred.RefreshToken
	}
	return parsed.AccessToken, refresh, expires, nil
}

func refreshPinterestOAuth(ctx context.Context, client *http.Client, tokenURL, clientID, clientSecret string, cred Credential) (string, string, time.Time, error) {
	if cred.RefreshToken == "" {
		return "", "", time.Time{}, fmt.Errorf("pinterest oauth: missing refresh token")
	}
	if clientID == "" || clientSecret == "" {
		return "", "", time.Time{}, fmt.Errorf("pinterest oauth: missing client id or secret")
	}
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	endpoint := tokenURL
	if endpoint == "" {
		endpoint = "https://api.pinterest.com/v5/oauth/token"
	}

	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", cred.RefreshToken)
	form.Set("client_id", clientID)
	form.Set("client_secret", clientSecret)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return "", "", time.Time{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

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
		return "", "", time.Time{}, fmt.Errorf("pinterest oauth refresh: read body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", "", time.Time{}, fmt.Errorf("pinterest oauth refresh: status %d: %s", resp.StatusCode, string(body))
	}

	var parsed struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int64  `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", "", time.Time{}, err
	}
	if parsed.AccessToken == "" {
		return "", "", time.Time{}, fmt.Errorf("pinterest oauth refresh: empty access_token")
	}
	expiresIn := parsed.ExpiresIn
	if expiresIn <= 0 {
		expiresIn = 3600
	}
	expires := time.Now().Add(time.Duration(expiresIn) * time.Second)
	refresh := parsed.RefreshToken
	if refresh == "" {
		refresh = cred.RefreshToken
	}
	return parsed.AccessToken, refresh, expires, nil
}

func refreshTrafficStarsOAuth(ctx context.Context, client *http.Client, baseURL string, cred Credential) (string, time.Time, error) {
	offlineKey := strings.TrimSpace(cred.RefreshToken)
	if offlineKey == "" {
		offlineKey = strings.TrimSpace(cred.APIKey)
	}
	if offlineKey == "" {
		return "", time.Time{}, fmt.Errorf("trafficstars oauth: missing offline api key")
	}
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	base := baseURL
	if base == "" {
		base = "https://api.trafficstars.com"
	}

	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", offlineKey)

	endpoint := strings.TrimRight(base, "/") + "/v1/auth/token"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
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
		return "", time.Time{}, fmt.Errorf("trafficstars oauth refresh: read body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", time.Time{}, fmt.Errorf("trafficstars oauth refresh: status %d: %s", resp.StatusCode, string(body))
	}

	var parsed struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int64  `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", time.Time{}, err
	}
	if parsed.AccessToken == "" {
		return "", time.Time{}, fmt.Errorf("trafficstars oauth refresh: empty access_token")
	}
	expiresIn := parsed.ExpiresIn
	if expiresIn <= 0 {
		expiresIn = 36000
	}
	expires := time.Now().Add(time.Duration(expiresIn) * time.Second)
	return parsed.AccessToken, expires, nil
}

func refreshRevcontentOAuth(ctx context.Context, client *http.Client, baseURL string, cred Credential) (string, time.Time, error) {
	clientID := strings.TrimSpace(cred.AccountID)
	if clientID == "" {
		clientID = strings.TrimSpace(cred.ExtraConfig["client_id"])
	}
	clientSecret := strings.TrimSpace(cred.APIKey)
	if clientSecret == "" {
		clientSecret = strings.TrimSpace(cred.ExtraConfig["client_secret"])
	}
	if clientID == "" || clientSecret == "" {
		return "", time.Time{}, fmt.Errorf("revcontent oauth: missing client_id or client_secret")
	}
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	base := baseURL
	if base == "" {
		base = "https://api.revcontent.io"
	}

	form := url.Values{}
	form.Set("grant_type", "client_credentials")
	form.Set("client_id", clientID)
	form.Set("client_secret", clientSecret)

	endpoint := strings.TrimRight(base, "/") + "/oauth/token"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
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
		return "", time.Time{}, fmt.Errorf("revcontent oauth: read body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", time.Time{}, fmt.Errorf("revcontent oauth: status %d: %s", resp.StatusCode, string(body))
	}

	var parsed struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int64  `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", time.Time{}, err
	}
	if parsed.AccessToken == "" {
		return "", time.Time{}, fmt.Errorf("revcontent oauth: empty access_token")
	}
	expiresIn := parsed.ExpiresIn
	if expiresIn <= 0 {
		expiresIn = 24 * 3600
	}
	expires := time.Now().Add(time.Duration(expiresIn) * time.Second)
	return parsed.AccessToken, expires, nil
}
