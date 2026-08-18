package controlplane

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/bidshard/ad-event-processor/pkg/coldpath"
)

const (
	cloudflareDefaultBase   = "https://api.cloudflare.com/client/v4"
	cloudflareMaxJSONBytes  = 1 << 20
	cloudflareClientTimeout = 10 * time.Second
)

type CloudflareZone struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type CloudflareAPI interface {
	ListZones(ctx context.Context) ([]CloudflareZone, error)
	CreateDNSRecord(ctx context.Context, zoneID, name, recordType, content string, proxied bool) (recordID string, err error)
	ZoneSSLStatus(ctx context.Context, zoneID string) (status string, err error)
}

type cloudflareClient struct {
	baseURL string
	token   string
	http    *http.Client
}

func NewCloudflareClient(token, baseURL string) CloudflareAPI {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil
	}
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if base == "" {
		base = cloudflareDefaultBase
	}
	return &cloudflareClient{
		baseURL: base,
		token:   token,
		http: &http.Client{
			Timeout: cloudflareClientTimeout,
		},
	}
}

type cloudflareResponse struct {
	Success bool              `json:"success"`
	Errors  []cloudflareError `json:"errors"`
	Result  json.RawMessage   `json:"result"`
}

type cloudflareError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (c *cloudflareClient) ListZones(ctx context.Context) ([]CloudflareZone, error) {
	if c == nil {
		return nil, errors.New("cloudflare client unavailable")
	}
	body, err := c.do(ctx, http.MethodGet, "/zones", nil)
	if err != nil {
		return nil, err
	}
	var zones []CloudflareZone
	if err := json.Unmarshal(body, &zones); err != nil {
		return nil, fmt.Errorf("cloudflare list zones decode: %w", err)
	}
	return zones, nil
}

func (c *cloudflareClient) CreateDNSRecord(ctx context.Context, zoneID, name, recordType, content string, proxied bool) (string, error) {
	if c == nil {
		return "", errors.New("cloudflare client unavailable")
	}
	zoneID = strings.TrimSpace(zoneID)
	if zoneID == "" {
		return "", errors.New("cloudflare zone id required")
	}
	payload := map[string]any{
		"type":    strings.ToUpper(strings.TrimSpace(recordType)),
		"name":    strings.TrimSpace(name),
		"content": strings.TrimSpace(content),
		"proxied": proxied,
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	body, err := c.do(ctx, http.MethodPost, "/zones/"+zoneID+"/dns_records", raw)
	if err != nil {
		return "", err
	}
	var rec struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(body, &rec); err != nil {
		return "", fmt.Errorf("cloudflare create dns decode: %w", err)
	}
	if rec.ID == "" {
		return "", errors.New("cloudflare create dns: empty record id")
	}
	return rec.ID, nil
}

func (c *cloudflareClient) ZoneSSLStatus(ctx context.Context, zoneID string) (string, error) {
	if c == nil {
		return "", errors.New("cloudflare client unavailable")
	}
	zoneID = strings.TrimSpace(zoneID)
	if zoneID == "" {
		return "", errors.New("cloudflare zone id required")
	}
	body, err := c.do(ctx, http.MethodGet, "/zones/"+zoneID+"/settings/ssl", nil)
	if err != nil {
		return "", err
	}
	var setting struct {
		Value string `json:"value"`
	}
	if err := json.Unmarshal(body, &setting); err != nil {
		return "", fmt.Errorf("cloudflare ssl status decode: %w", err)
	}
	if setting.Value == "" {
		return "unknown", nil
	}
	return setting.Value, nil
}

func (c *cloudflareClient) do(ctx context.Context, method, path string, body []byte) (json.RawMessage, error) {
	var r io.Reader
	if len(body) > 0 {
		r = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, r)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		coldpath.CloseHTTPResponse(resp)
		return nil, err
	}
	defer coldpath.CloseHTTPResponse(resp)

	limited := io.LimitReader(resp.Body, cloudflareMaxJSONBytes)
	raw, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("cloudflare http %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}

	var envelope cloudflareResponse
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	if err := dec.Decode(&envelope); err != nil {
		return nil, fmt.Errorf("cloudflare envelope decode: %w", err)
	}
	if !envelope.Success {
		if len(envelope.Errors) > 0 {
			return nil, fmt.Errorf("cloudflare api: %s", envelope.Errors[0].Message)
		}
		return nil, errors.New("cloudflare api: request failed")
	}
	return envelope.Result, nil
}

func cloudflareRecordTypeForTarget(target string) string {
	target = strings.TrimSpace(target)
	if target == "" {
		return "A"
	}
	if strings.Contains(target, ":") {
		return "AAAA"
	}
	for i := 0; i < len(target); i++ {
		c := target[i]
		if (c < '0' || c > '9') && c != '.' {
			return "CNAME"
		}
	}
	return "A"
}
