package postback

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/bidshard/ad-event-processor/pkg/money"

	"github.com/google/uuid"
)

// DryRunResult is the structured admin test response for a postback/CAPI config.
type DryRunResult struct {
	OK          bool   `json:"ok"`
	Provider    string `json:"provider"`
	HTTPStatus  int    `json:"http_status,omitempty"`
	Error       string `json:"error,omitempty"`
	RenderedURL string `json:"rendered_url,omitempty"`
	TargetEvent string `json:"target_event,omitempty"`
	TestEvent   bool   `json:"test_event"`
}

func postbackAdapters() map[string]PostbackAdapter {
	return map[string]PostbackAdapter{
		"facebook": &FacebookAdapter{},
		"google":   &GoogleAdapter{},
		"tiktok":   &TikTokAdapter{},
		"webhook":  &WebhookAdapter{},
	}
}

// DryRunConfig exercises the configured provider with a synthetic payload (no outbox write).
func DryRunConfig(
	ctx context.Context,
	provider, urlTemplate, apiToken, targetEvent, testEventCode string,
	campaignID uuid.UUID,
) DryRunResult {
	provider = strings.ToLower(strings.TrimSpace(provider))
	adapter := postbackAdapters()[provider]
	if adapter == nil {
		return DryRunResult{OK: false, Provider: provider, Error: "unsupported provider"}
	}
	if strings.TrimSpace(urlTemplate) == "" {
		return DryRunResult{OK: false, Provider: provider, Error: "url_template required"}
	}
	if provider != "webhook" && strings.TrimSpace(apiToken) == "" {
		return DryRunResult{OK: false, Provider: provider, Error: "api_token required for CAPI providers"}
	}
	if targetEvent == "" {
		targetEvent = "conversion"
	}
	payload := &PostbackPayload{
		CampaignID:     campaignID,
		ClickID:        "dry-run-click-id",
		EventType:      targetEvent,
		PayoutMicro:    1_000_000,
		TxID:           "dry-run-tx",
		SubID1:         "dry-run-sub",
		FBCLID:         "dry-run-fbclid",
		GCLID:          "dry-run-gclid",
		TTCLID:         "dry-run-ttclid",
		EventSourceURL: "https://example.com/lp",
		TestEventCode:  strings.TrimSpace(testEventCode),
	}
	client := &http.Client{Timeout: 8 * time.Second}
	err := adapter.Send(ctx, client, payload, urlTemplate, apiToken)
	if err == nil {
		rendered := ""
		if provider == "webhook" {
			mt := (&WebhookAdapter{}).cachedTemplate(urlTemplate)
			evtCtx := &EventContext{
				ClickID:   payload.ClickID,
				Payout:    money.FormatDecimal(payload.PayoutMicro),
				TxID:      payload.TxID,
				EventType: payload.EventType,
			}
			evtCtx.SubIDs = payload.SubIDs()
			var scratch [MaxRenderedURLLen]byte
			rendered = string(mt.RenderStack(evtCtx, &scratch))
		}
		return DryRunResult{
			OK:          true,
			Provider:    provider,
			HTTPStatus:  http.StatusOK,
			RenderedURL: rendered,
			TargetEvent: targetEvent,
			TestEvent:   payload.TestEventCode != "",
		}
	}
	res := DryRunResult{
		OK:          false,
		Provider:    provider,
		Error:       err.Error(),
		TargetEvent: targetEvent,
		TestEvent:   payload.TestEventCode != "",
	}
	var httpErr *DispatchHTTPError
	if errors.As(err, &httpErr) {
		res.HTTPStatus = httpErr.StatusCode
	}
	return res
}
