package ivtdetector

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type BlacklistBlocker interface {
	BlockIP(ctx context.Context, ip string) error
	EnqueueFraudThreat(ctx context.Context, action string, ip string, campaignID string, score float64, boost int32, ttlSeconds int64) error
}

const blacklistSourceFraud = "fraud"

type ManagementClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

func NewManagementClient(baseURL, apiKey string, timeout time.Duration) *ManagementClient {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	return &ManagementClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func (client *ManagementClient) BlockIP(ctx context.Context, ip string) error {
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
		return fmt.Errorf("%w: %v", ErrManagementUnavailable, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusCreated {
		return nil
	}

	payload, readErr := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if readErr != nil {
		return fmt.Errorf("%w: status=%d read body: %v", ErrManagementUnavailable, resp.StatusCode, readErr)
	}
	return fmt.Errorf("%w: status=%d body=%s", ErrManagementUnavailable, resp.StatusCode, strings.TrimSpace(string(payload)))
}

func (client *ManagementClient) EnqueueFraudThreat(ctx context.Context, action string, ip string, campaignID string, score float64, boost int32, ttlSeconds int64) error {
	return fmt.Errorf("EnqueueFraudThreat not implemented on HTTP client; use gRPC client")
}
