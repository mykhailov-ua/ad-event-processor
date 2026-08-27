package fraud

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"ad-event-processor/pkg/coldpath"
)

type BlacklistBlocker interface {
	BlockIP(ctx context.Context, ip string) error
	EnqueueFraudThreat(ctx context.Context, action string, ip string, campaignID string, score float64, boost int32, ttlSeconds int64) error
	EnqueueFraudThreatBatch(ctx context.Context, items []FraudThreatEnqueueItem) (int, error)
}

const blacklistSourceFraud = "fraud"

type ControlplaneClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

func NewControlplaneClient(baseURL, apiKey string, timeout time.Duration) *ControlplaneClient {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	return &ControlplaneClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func (c *ControlplaneClient) BlockIP(ctx context.Context, ip string) error {
	if c == nil {
		return fmt.Errorf("management client: nil receiver")
	}
	if ip == "" {
		return ErrInvalidIP
	}

	body, err := json.Marshal(map[string]string{
		"ip":     ip,
		"source": blacklistSourceFraud,
	})
	if err != nil {
		return fmt.Errorf("marshal blacklist request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/ops/blacklist", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build blacklist request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Admin-API-Key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		coldpath.CloseHTTPResponse(resp)
		return fmt.Errorf("%w: %w", ErrManagementUnavailable, err)
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()

	if resp.StatusCode == http.StatusCreated {
		return nil
	}

	payload, readErr := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if readErr != nil {
		return fmt.Errorf("%w: status=%d read body: %w", ErrManagementUnavailable, resp.StatusCode, readErr)
	}
	return fmt.Errorf("%w: status=%d body=%s", ErrManagementUnavailable, resp.StatusCode, strings.TrimSpace(string(payload)))
}

func (c *ControlplaneClient) EnqueueFraudThreat(ctx context.Context, action, ip, campaignID string, score float64, boost int32, ttlSeconds int64) error {
	_, err := c.EnqueueFraudThreatBatch(ctx, []FraudThreatEnqueueItem{{
		Action:     action,
		IP:         ip,
		CampaignID: campaignID,
		Score:      score,
		Boost:      boost,
		TTLSeconds: ttlSeconds,
	}})
	return err
}

func (c *ControlplaneClient) EnqueueFraudThreatBatch(ctx context.Context, items []FraudThreatEnqueueItem) (int, error) {
	if c == nil {
		return 0, fmt.Errorf("management client: nil receiver")
	}
	if len(items) == 0 {
		return 0, fmt.Errorf("invalid fraud threat batch request")
	}

	body, err := json.Marshal(fraudThreatBatchRequest{Items: items})
	if err != nil {
		return 0, fmt.Errorf("marshal fraud threat batch request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/ops/fraud-threat", bytes.NewReader(body))
	if err != nil {
		return 0, fmt.Errorf("build fraud threat batch request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Admin-API-Key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		coldpath.CloseHTTPResponse(resp)
		return 0, fmt.Errorf("%w: %w", ErrManagementUnavailable, err)
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()

	if resp.StatusCode == http.StatusOK {
		return len(items), nil
	}

	payload, readErr := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if readErr != nil {
		return 0, fmt.Errorf("%w: status=%d read body: %w", ErrManagementUnavailable, resp.StatusCode, readErr)
	}
	return 0, fmt.Errorf("%w: status=%d body=%s", ErrManagementUnavailable, resp.StatusCode, strings.TrimSpace(string(payload)))
}
