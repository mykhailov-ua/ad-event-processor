package postback

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

const outbrainS2SDefaultURL = "https://tr.outbrain.com/unifiedPixel"

type OutbrainAdapter struct{}

func (a *OutbrainAdapter) Send(ctx context.Context, client *http.Client, payload *PostbackPayload, urlTemplate, apiTokenDecrypted string) error {
	clickID := resolveOutbrainClickID(payload)
	if clickID == "" {
		return fmt.Errorf("outbrain: missing ob_click_id on conversion payload")
	}

	eventName := resolveOutboundEventName(urlTemplate, payload.EventType, "conversion")
	if eventName == "" {
		return fmt.Errorf("outbrain: missing conversion event name (url_template or event_type)")
	}

	endpoint := outbrainS2SDefaultURL
	if t := strings.TrimSpace(urlTemplate); strings.HasPrefix(t, "http") {
		endpoint = t
	}

	q := url.Values{}
	q.Set("ob_click_id", clickID)
	q.Set("name", eventName)
	if payload.PayoutMicro > 0 {
		q.Set("orderValue", fmt.Sprintf("%.2f", payload.PayoutDollarsAPI()))
		q.Set("currency", "USD")
	}

	reqURL := endpoint + "?" + q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, http.NoBody)
	if err != nil {
		return fmt.Errorf("outbrain: create request: %w", err)
	}
	if apiTokenDecrypted != "" {
		req.Header.Set("Authorization", "Bearer "+apiTokenDecrypted)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("outbrain: http request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	return checkHTTPResponse(resp)
}
