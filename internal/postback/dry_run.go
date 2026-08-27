package postback

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"ad-event-processor/pkg/money"

	"github.com/google/uuid"
)

type DryRunResult struct {
	OK          bool     `json:"ok"`
	Provider    string   `json:"provider"`
	HTTPStatus  int      `json:"http_status,omitempty"`
	Error       string   `json:"error,omitempty"`
	RenderedURL string   `json:"rendered_url,omitempty"`
	TargetEvent string   `json:"target_event,omitempty"`
	TestEvent   bool     `json:"test_event"`
	Warnings    []string `json:"warnings,omitempty"`
}

func postbackAdapters() map[string]PostbackAdapter {
	return map[string]PostbackAdapter{
		"facebook":      &FacebookAdapter{},
		"google":        &GoogleAdapter{},
		"tiktok":        &TikTokAdapter{},
		"taboola":       &TaboolaAdapter{},
		"outbrain":      &OutbrainAdapter{},
		"microsoft_ads": &MicrosoftAdsAdapter{},
		"webhook":       &WebhookAdapter{},
	}
}

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
	if ProviderRequiresToken(provider) && strings.TrimSpace(apiToken) == "" {
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
		EventID:        "dry-run-event-id",
		SubID1:         "dry-run-sub",
		FBCLID:         "dry-run-fbclid",
		GCLID:          "dry-run-gclid",
		TTCLID:         "dry-run-ttclid",
		TBLCI:          "dry-run-tblci",
		OBClickID:      "dry-run-ob-click",
		MSCLKID:        "dry-run-msclkid",
		EventSourceURL: "https://example.com/lp",
		TestEventCode:  strings.TrimSpace(testEventCode),
	}
	warnings := PostbackClickIDWarnings(provider, payload)
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
			Warnings:    warnings,
		}
	}
	res := DryRunResult{
		OK:          false,
		Provider:    provider,
		Error:       err.Error(),
		TargetEvent: targetEvent,
		TestEvent:   payload.TestEventCode != "",
		Warnings:    warnings,
	}
	var httpErr *DispatchHTTPError
	if errors.As(err, &httpErr) {
		res.HTTPStatus = httpErr.StatusCode
	}
	return res
}
