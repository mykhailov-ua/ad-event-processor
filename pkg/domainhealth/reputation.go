package domainhealth

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
)

const (
	reputationTimeout       = 8 * time.Second
	reputationMaxJSONBytes  = 1 << 20
	defaultFacebookGraphAPI = "https://graph.facebook.com/v19.0"
	reputationClientID      = "bidshard"
	reputationClientVersion = "1.0"
)

type ReputationConfig struct {
	SafeBrowsingAPIKey string
	SafeBrowsingAPIURL string
	FacebookToken      string
	FacebookGraphBase  string
}

type ReputationChecker struct {
	cfg  ReputationConfig
	http *http.Client
}

func NewReputationChecker(cfg ReputationConfig) *ReputationChecker {
	cfg.SafeBrowsingAPIKey = strings.TrimSpace(cfg.SafeBrowsingAPIKey)
	cfg.FacebookToken = strings.TrimSpace(cfg.FacebookToken)
	cfg.FacebookGraphBase = strings.TrimRight(strings.TrimSpace(cfg.FacebookGraphBase), "/")
	if cfg.FacebookGraphBase == "" {
		cfg.FacebookGraphBase = defaultFacebookGraphAPI
	}
	if cfg.SafeBrowsingAPIKey == "" && cfg.FacebookToken == "" {
		return nil
	}
	return &ReputationChecker{
		cfg:  cfg,
		http: &http.Client{Timeout: reputationTimeout},
	}
}

func (r *ReputationChecker) Enabled() bool {
	return r != nil
}

func (r *ReputationChecker) Check(ctx context.Context, hostname string) (unsafe bool, detail string, err error) {
	if r == nil {
		return false, "", nil
	}
	host := strings.TrimSpace(hostname)
	if host == "" {
		return false, "", nil
	}
	pageURL := "https://" + host + "/"

	if r.cfg.SafeBrowsingAPIKey != "" {
		flagged, why, err := r.checkSafeBrowsing(ctx, pageURL)
		if err != nil {
			return false, "", err
		}
		if flagged {
			return true, "safe_browsing:" + why, nil
		}
	}

	if r.cfg.FacebookToken != "" {
		flagged, why, err := r.checkFacebookScrape(ctx, pageURL)
		if err != nil {
			return false, "", err
		}
		if flagged {
			return true, "facebook_graph:" + why, nil
		}
	}

	return false, "", nil
}

type safeBrowsingRequest struct {
	Client     safeBrowsingClient     `json:"client"`
	ThreatInfo safeBrowsingThreatInfo `json:"threatInfo"`
}

type safeBrowsingClient struct {
	ClientID      string `json:"clientId"`
	ClientVersion string `json:"clientVersion"`
}

type safeBrowsingThreatInfo struct {
	ThreatTypes      []string               `json:"threatTypes"`
	PlatformTypes    []string               `json:"platformTypes"`
	ThreatEntryTypes []string               `json:"threatEntryTypes"`
	ThreatEntries    []safeBrowsingURLEntry `json:"threatEntries"`
}

type safeBrowsingURLEntry struct {
	URL string `json:"url"`
}

type safeBrowsingMatch struct {
	ThreatType string `json:"threatType"`
}

type safeBrowsingResponse struct {
	Matches []safeBrowsingMatch `json:"matches"`
}

func (r *ReputationChecker) checkSafeBrowsing(ctx context.Context, pageURL string) (bool, string, error) {
	body, err := json.Marshal(safeBrowsingRequest{
		Client: safeBrowsingClient{ClientID: reputationClientID, ClientVersion: reputationClientVersion},
		ThreatInfo: safeBrowsingThreatInfo{
			ThreatTypes: []string{
				"MALWARE",
				"SOCIAL_ENGINEERING",
				"UNWANTED_SOFTWARE",
				"POTENTIALLY_HARMFUL_APPLICATION",
			},
			PlatformTypes:    []string{"ANY_PLATFORM"},
			ThreatEntryTypes: []string{"URL"},
			ThreatEntries:    []safeBrowsingURLEntry{{URL: pageURL}},
		},
	})
	if err != nil {
		return false, "", err
	}
	endpoint := strings.TrimSpace(r.cfg.SafeBrowsingAPIURL)
	if endpoint == "" {
		endpoint = "https://safebrowsing.googleapis.com/v4/threatMatches:find?key=" + url.QueryEscape(r.cfg.SafeBrowsingAPIKey)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return false, "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.http.Do(req)
	if err != nil {
		return false, "", err
	}
	defer func() { _ = resp.Body.Close() }()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, reputationMaxJSONBytes))
	if err != nil {
		return false, "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return false, "", fmt.Errorf("safe browsing http %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}

	var out safeBrowsingResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return false, "", fmt.Errorf("safe browsing decode: %w", err)
	}
	if len(out.Matches) == 0 {
		return false, "", nil
	}
	threat := out.Matches[0].ThreatType
	if threat == "" {
		threat = "match"
	}
	return true, threat, nil
}

type facebookScrapeResponse struct {
	IsBlocked bool `json:"is_blocked"`
}

func (r *ReputationChecker) checkFacebookScrape(ctx context.Context, pageURL string) (bool, string, error) {
	u, err := url.Parse(r.cfg.FacebookGraphBase + "/")
	if err != nil {
		return false, "", err
	}
	q := u.Query()
	q.Set("id", pageURL)
	q.Set("scrape", "true")
	q.Set("access_token", r.cfg.FacebookToken)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), http.NoBody)
	if err != nil {
		return false, "", err
	}
	resp, err := r.http.Do(req)
	if err != nil {
		return false, "", err
	}
	defer func() { _ = resp.Body.Close() }()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, reputationMaxJSONBytes))
	if err != nil {
		return false, "", err
	}
	if resp.StatusCode >= 400 {
		msg := strings.ToLower(string(raw))
		if strings.Contains(msg, "blocked") || strings.Contains(msg, "unsafe") {
			return true, "graph_error", nil
		}
		return false, "", fmt.Errorf("facebook graph http %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}

	var out facebookScrapeResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return false, "", fmt.Errorf("facebook graph decode: %w", err)
	}
	if out.IsBlocked {
		return true, "is_blocked", nil
	}
	return false, "", nil
}
