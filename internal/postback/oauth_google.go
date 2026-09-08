package postback

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

type googleOAuthCredentials struct {
	AccessToken     string
	RefreshToken    string
	ClientID        string
	ClientSecret    string
	DeveloperToken  string
	LoginCustomerID string
	ExpiresAt       time.Time
}

func parseGoogleOAuthCredentials(apiTokenDecrypted, developerTokenFromConfig string) googleOAuthCredentials {
	raw := strings.TrimSpace(apiTokenDecrypted)
	out := googleOAuthCredentials{
		DeveloperToken: strings.TrimSpace(developerTokenFromConfig),
	}
	if raw == "" {
		return out
	}
	if strings.HasPrefix(raw, "{") {
		var parsed struct {
			AccessToken     string `json:"access_token"`
			RefreshToken    string `json:"refresh_token"`
			ClientID        string `json:"client_id"`
			ClientSecret    string `json:"client_secret"`
			DeveloperToken  string `json:"developer_token"`
			LoginCustomerID string `json:"login_customer_id"`
			ExpiresAt       string `json:"expires_at"`
		}
		if json.Unmarshal([]byte(raw), &parsed) == nil {
			out.AccessToken = strings.TrimSpace(parsed.AccessToken)
			out.RefreshToken = strings.TrimSpace(parsed.RefreshToken)
			out.ClientID = strings.TrimSpace(parsed.ClientID)
			out.ClientSecret = strings.TrimSpace(parsed.ClientSecret)
			if dev := strings.TrimSpace(parsed.DeveloperToken); dev != "" {
				out.DeveloperToken = dev
			}
			out.LoginCustomerID = strings.TrimSpace(parsed.LoginCustomerID)
			if parsed.ExpiresAt != "" {
				if expires, err := time.Parse(time.RFC3339, parsed.ExpiresAt); err == nil {
					out.ExpiresAt = expires
				}
			}
			return out
		}
	}
	out.AccessToken = raw
	return out
}

func refreshGoogleOAuth(
	ctx context.Context,
	client *http.Client,
	clientID, clientSecret, refreshToken string,
) (string, time.Time, error) {
	if refreshToken == "" {
		return "", time.Time{}, fmt.Errorf("google oauth: missing refresh token")
	}
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}

	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", refreshToken)
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
		return "", time.Time{}, fmt.Errorf("google oauth refresh: read body: %w", err)
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
	return strings.TrimSpace(parsed.AccessToken), expires, nil
}

func resolveGoogleAccessToken(
	ctx context.Context,
	client *http.Client,
	apiTokenDecrypted, developerTokenFromConfig string,
) (accessToken, developerToken, loginCustomerID string, err error) {
	cred := parseGoogleOAuthCredentials(apiTokenDecrypted, developerTokenFromConfig)
	access := cred.AccessToken
	needsRefresh := access == "" ||
		(cred.RefreshToken != "" && cred.ClientID != "" && cred.ClientSecret != "" &&
			!cred.ExpiresAt.IsZero() && time.Now().After(cred.ExpiresAt.Add(-30*time.Second)))
	if needsRefresh && cred.RefreshToken != "" && cred.ClientID != "" && cred.ClientSecret != "" {
		refreshed, _, refreshErr := refreshGoogleOAuth(ctx, client, cred.ClientID, cred.ClientSecret, cred.RefreshToken)
		if refreshErr != nil {
			if access == "" {
				return "", cred.DeveloperToken, cred.LoginCustomerID, refreshErr
			}
		} else if refreshed != "" {
			access = refreshed
		}
	}
	if access == "" {
		return "", cred.DeveloperToken, cred.LoginCustomerID, fmt.Errorf("google: oauth access token required")
	}
	return access, cred.DeveloperToken, cred.LoginCustomerID, nil
}
