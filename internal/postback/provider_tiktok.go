package postback

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type TikTokAdapter struct{}

type TikTokUserContext struct {
	Ttclid string `json:"ttclid,omitempty"`
	Email  string `json:"email,omitempty"`
	Phone  string `json:"phone,omitempty"`
}

type TikTokEvent struct {
	Event     string            `json:"event"`
	EventID   string            `json:"event_id,omitempty"`
	Timestamp string            `json:"timestamp"`
	Context   TikTokUserContext `json:"context"`
	Value     float64           `json:"value,omitempty"`
	Currency  string            `json:"currency,omitempty"`
}

type TikTokCAPIPayload struct {
	PixelCode     string        `json:"pixel_code,omitempty"`
	TestEventCode string        `json:"test_event_code,omitempty"`
	Events        []TikTokEvent `json:"events"`
}

func mapTikTokEventName(eventType string) string {
	switch strings.ToLower(strings.TrimSpace(eventType)) {
	case "lead":
		return "Contact"
	case "click":
		return "ClickButton"
	case "install":
		return "Download"
	default:
		return "CompletePayment"
	}
}

func resolveTikTokPixelCode(urlTemplate string) string {
	if urlTemplate == "" || strings.HasPrefix(urlTemplate, "http") {
		return ""
	}
	return urlTemplate
}

func (a *TikTokAdapter) Send(ctx context.Context, client *http.Client, payload *PostbackPayload, urlTemplate, apiTokenDecrypted string) error {
	url := urlTemplate
	if url == "" || !strings.HasPrefix(url, "http") {
		url = "https://business-api.tiktok.com/open_api/v1.3/event/track/"
	}

	pixelCode := resolveTikTokPixelCode(urlTemplate)

	event := mapTikTokEventName(payload.EventType)

	hashedEmail := ""
	if payload.Email != "" {
		hashedEmail = hashSHA256(payload.Email)
	}
	hashedPhone := ""
	if payload.Phone != "" {
		hashedPhone = hashSHA256(payload.Phone)
	}

	ttEvent := TikTokEvent{
		Event:     event,
		EventID:   payload.ClickID,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Context: TikTokUserContext{
			Ttclid: payload.TTCLID,
			Email:  hashedEmail,
			Phone:  hashedPhone,
		},
	}

	if payload.PayoutMicro > 0 {
		ttEvent.Value = payload.PayoutDollarsAPI()
		ttEvent.Currency = "USD"
	}

	tiktokPayload := TikTokCAPIPayload{
		PixelCode:     pixelCode,
		TestEventCode: strings.TrimSpace(payload.TestEventCode),
		Events:        []TikTokEvent{ttEvent},
	}

	bodyBytes, err := json.Marshal(tiktokPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if apiTokenDecrypted != "" {
		req.Header.Set("Access-Token", apiTokenDecrypted)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("http request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return checkHTTPResponse(resp)
	}

	return nil
}
