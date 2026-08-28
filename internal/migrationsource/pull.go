package migrationsource

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"ad-event-processor/pkg/branding"
)

const migrationPullTimeout = 30 * time.Second

type PullSpec struct {
	SourceKind SourceKind
	BaseURL    string
	APIToken   string
	PullPath   string
}

var migrationPullHTTPClient = &http.Client{Timeout: migrationPullTimeout}

func FetchRemotePayload(ctx context.Context, spec PullSpec) ([]byte, error) {
	spec.BaseURL = strings.TrimSpace(spec.BaseURL)
	spec.APIToken = strings.TrimSpace(spec.APIToken)
	spec.PullPath = strings.TrimSpace(spec.PullPath)
	if spec.BaseURL == "" {
		return nil, fmt.Errorf("base_url is required")
	}
	if spec.APIToken == "" {
		return nil, fmt.Errorf("api_token is required")
	}
	switch spec.SourceKind {
	case SourceKindKeitaroAdminAPI:
		return pullKeitaroAdminAPI(ctx, spec)
	case SourceKindBinomReportAPI:
		return pullBinomReportAPI(ctx, spec)
	default:
		return nil, fmt.Errorf("source_kind %q does not support live pull", spec.SourceKind)
	}
}

func pullKeitaroAdminAPI(ctx context.Context, spec PullSpec) ([]byte, error) {
	path := spec.PullPath
	if path == "" {
		path = "/admin_api/v1/campaigns"
	}
	endpoint, err := joinMigrationPullURL(spec.BaseURL, path)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Api-Key", spec.APIToken)
	req.Header.Set("User-Agent", branding.HTTPUserAgent("MigrationPull"))
	return doMigrationPullRequest(req)
}

func pullBinomReportAPI(ctx context.Context, spec PullSpec) ([]byte, error) {
	path := spec.PullPath
	if path == "" {
		path = "/public/api/v1/campaign/list"
	}
	endpoint, err := joinMigrationPullURL(spec.BaseURL, path)
	if err != nil {
		return nil, err
	}
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}
	q := parsed.Query()
	q.Set("api_key", spec.APIToken)
	parsed.RawQuery = q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", branding.HTTPUserAgent("MigrationPull"))
	body, err := doMigrationPullRequest(req)
	if err != nil {
		return nil, err
	}
	return normalizeBinomPullBody(body)
}

func doMigrationPullRequest(req *http.Request) ([]byte, error) {
	resp, err := migrationPullHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("migration pull request failed: %w", err)
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("migration pull returned http %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, MaxPayloadBytes))
	if err != nil {
		return nil, err
	}
	if len(bytesTrimSpace(body)) == 0 {
		return nil, fmt.Errorf("migration pull returned empty body")
	}
	return body, nil
}

func joinMigrationPullURL(base, path string) (string, error) {
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	path = "/" + strings.TrimLeft(strings.TrimSpace(path), "/")
	if base == "" {
		return "", fmt.Errorf("base_url is required")
	}
	parsed, err := url.Parse(base + path)
	if err != nil {
		return "", err
	}
	if parsed.Scheme != "https" && parsed.Scheme != "http" {
		return "", fmt.Errorf("base_url must be http or https")
	}
	return parsed.String(), nil
}

func normalizeBinomPullBody(body []byte) ([]byte, error) {
	body = bytesTrimSpace(body)
	if len(body) == 0 {
		return nil, fmt.Errorf("empty binom pull body")
	}
	if body[0] == '[' {
		return body, nil
	}
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(body, &doc); err != nil {
		return body, nil
	}
	for _, key := range []string{"rows", "campaigns", "data"} {
		if raw, ok := doc[key]; ok && len(bytesTrimSpace(raw)) > 0 {
			return bytesTrimSpace(raw), nil
		}
	}
	return body, nil
}

func PullSupported(kind SourceKind) bool {
	switch kind {
	case SourceKindKeitaroAdminAPI, SourceKindBinomReportAPI:
		return true
	default:
		return false
	}
}
