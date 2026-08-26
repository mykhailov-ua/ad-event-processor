package postback

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

const taboolaS2SDefaultURL = "https://trc.taboola.com/actions-handler/log/3/s2s-action"

type TaboolaAdapter struct{}

func (a *TaboolaAdapter) Send(ctx context.Context, client *http.Client, payload *PostbackPayload, urlTemplate, apiTokenDecrypted string) error {
	clickID := resolveTaboolaClickID(payload)
	if clickID == "" {
		return fmt.Errorf("taboola: missing click-id (tblci on conversion payload)")
	}

	eventName := resolveOutboundEventName(urlTemplate, payload.EventType, "conversion")
	if eventName == "" {
		return fmt.Errorf("taboola: missing event name (url_template or event_type)")
	}

	endpoint := taboolaS2SDefaultURL
	if t := strings.TrimSpace(urlTemplate); strings.HasPrefix(t, "http") {
		endpoint = t
	}

	q := url.Values{}
	q.Set("click-id", clickID)
	q.Set("name", eventName)
	if payload.PayoutMicro > 0 {
		q.Set("revenue", fmt.Sprintf("%.2f", payload.PayoutDollarsAPI()))
		q.Set("currency", "USD")
	}
	if orderID := strings.TrimSpace(payload.TxID); orderID != "" {
		q.Set("orderid", orderID)
	}

	var reqURL string
	if strings.Contains(endpoint, "?") {
		reqURL = endpoint + "&" + q.Encode()
	} else {
		reqURL = endpoint + "?" + q.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, http.NoBody)
	if err != nil {
		return fmt.Errorf("taboola: create request: %w", err)
	}
	if apiTokenDecrypted != "" {
		req.Header.Set("Authorization", "Bearer "+apiTokenDecrypted)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("taboola: http request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	return checkHTTPResponse(resp)
}
