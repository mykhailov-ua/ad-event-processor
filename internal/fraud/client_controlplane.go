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

	"github.com/bidshard/ad-event-processor/pkg/coldpath"
)

type BlacklistBlocker interface {
	BlockIP(ctx context.Context, ip string) error
	EnqueueFraudThreat(ctx context.Context, action string, ip string, campaignID string, score float64, boost int32, ttlSeconds int64) error
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

func (client *ControlplaneClient) BlockIP(ctx context.Context, ip string) error {
	if client == nil {
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

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, client.baseURL+"/api/v1/ops/blacklist", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build blacklist request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Admin-API-Key", client.apiKey)

	resp, err := client.httpClient.Do(req)
	if err != nil {
		coldpath.CloseHTTPResponse(resp)
		return fmt.Errorf("%w: %w", ErrManagementUnavailable, err)
	}
	defer coldpath.CloseHTTPResponse(resp)

	if resp.StatusCode == http.StatusCreated {
		return nil
	}

	payload, readErr := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if readErr != nil {
		return fmt.Errorf("%w: status=%d read body: %w", ErrManagementUnavailable, resp.StatusCode, readErr)
	}
	return fmt.Errorf("%w: status=%d body=%s", ErrManagementUnavailable, resp.StatusCode, strings.TrimSpace(string(payload)))
}

func (client *ControlplaneClient) EnqueueFraudThreat(ctx context.Context, action string, ip string, campaignID string, score float64, boost int32, ttlSeconds int64) error {
	if client == nil {
		return fmt.Errorf("management client: nil receiver")
	}
	if action == "" || campaignID == "" {
		return fmt.Errorf("invalid fraud threat request")
	}

	body, err := json.Marshal(map[string]any{
		"action":      action,
		"ip":          ip,
		"campaign_id": campaignID,
		"score":       score,
		"boost":       boost,
		"ttl_seconds": ttlSeconds,
	})
	if err != nil {
		return fmt.Errorf("marshal fraud threat request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, client.baseURL+"/api/v1/ops/fraud-threat", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build fraud threat request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Admin-API-Key", client.apiKey)

	resp, err := client.httpClient.Do(req)
	if err != nil {
		coldpath.CloseHTTPResponse(resp)
		return fmt.Errorf("%w: %w", ErrManagementUnavailable, err)
	}
	defer coldpath.CloseHTTPResponse(resp)

	if resp.StatusCode == http.StatusOK {
		return nil
	}

	payload, readErr := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if readErr != nil {
		return fmt.Errorf("%w: status=%d read body: %w", ErrManagementUnavailable, resp.StatusCode, readErr)
	}
	return fmt.Errorf("%w: status=%d body=%s", ErrManagementUnavailable, resp.StatusCode, strings.TrimSpace(string(payload)))
}
