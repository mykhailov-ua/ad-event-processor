package fraud

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

type ResidentialIntelProvider interface {
	Lookup(ctx context.Context, ip string) (ResidentialIntelResult, error)
}

type HTTPResidentialIntelProvider struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

func NewHTTPResidentialIntelProvider(baseURL, apiKey string, timeout time.Duration) (*HTTPResidentialIntelProvider, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return nil, fmt.Errorf("residential intel provider: empty base URL")
	}
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	return &HTTPResidentialIntelProvider{
		baseURL: baseURL,
		apiKey:  strings.TrimSpace(apiKey),
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}, nil
}

type residentialIntelAPIResponse = ResidentialIntelResult

func (p *HTTPResidentialIntelProvider) Lookup(ctx context.Context, ip string) (ResidentialIntelResult, error) {
	if p == nil {
		return ResidentialIntelResult{}, fmt.Errorf("residential intel provider: nil receiver")
	}
	ip = strings.TrimSpace(ip)
	if ip == "" {
		return ResidentialIntelResult{}, ErrInvalidIP
	}

	u, err := url.Parse(p.baseURL + "/lookup")
	if err != nil {
		return ResidentialIntelResult{}, fmt.Errorf("parse provider URL: %w", err)
	}
	q := u.Query()
	q.Set("ip", ip)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), http.NoBody)
	if err != nil {
		return ResidentialIntelResult{}, fmt.Errorf("build provider request: %w", err)
	}
	if p.apiKey != "" {
		req.Header.Set("X-API-Key", p.apiKey)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return ResidentialIntelResult{}, fmt.Errorf("provider request failed: %w", err)
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return ResidentialIntelResult{}, fmt.Errorf("provider status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, coldpath.DefaultMaxBody))
	if err != nil {
		return ResidentialIntelResult{}, fmt.Errorf("read provider body: %w", err)
	}
	var decoded residentialIntelAPIResponse
	if err := json.Unmarshal(body, &decoded); err != nil {
		return ResidentialIntelResult{}, fmt.Errorf("decode provider body: %w", err)
	}
	return decoded, nil
}

type StubResidentialIntelProvider struct {
	Results map[string]ResidentialIntelResult
}

func (p *StubResidentialIntelProvider) Lookup(_ context.Context, ip string) (ResidentialIntelResult, error) {
	if p == nil {
		return ResidentialIntelResult{}, fmt.Errorf("stub residential intel provider: nil receiver")
	}
	ip = strings.TrimSpace(ip)
	if ip == "" {
		return ResidentialIntelResult{}, ErrInvalidIP
	}
	if p.Results == nil {
		return ResidentialIntelResult{}, nil
	}
	if res, ok := p.Results[ip]; ok {
		return res, nil
	}
	return ResidentialIntelResult{}, nil
}
