package postback

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/bidshard/ad-event-processor/pkg/money"
)

type WebhookAdapter struct {
	templates sync.Map
}

func (a *WebhookAdapter) cachedTemplate(urlTemplate string) *MacroTemplate {
	if v, ok := a.templates.Load(urlTemplate); ok {
		return v.(*MacroTemplate)
	}
	mt := ParseTemplate(urlTemplate)
	if v, loaded := a.templates.LoadOrStore(urlTemplate, mt); loaded {
		return v.(*MacroTemplate)
	}
	return mt
}

func (a *WebhookAdapter) Send(ctx context.Context, client *http.Client, payload *PostbackPayload, urlTemplate string, apiTokenDecrypted string) error {
	mt := a.cachedTemplate(urlTemplate)
	evtCtx := &EventContext{
		ClickID:   payload.ClickID,
		Payout:    money.FormatDecimal(payload.PayoutMicro),
		TxID:      payload.TxID,
		EventType: payload.EventType,
	}
	evtCtx.SubIDs = payload.SubIDs()
	var scratch [MaxRenderedURLLen]byte
	renderedURL := string(mt.RenderStack(evtCtx, &scratch))

	req, err := http.NewRequestWithContext(ctx, "GET", renderedURL, http.NoBody)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if apiTokenDecrypted != "" {
		req.Header.Set("Authorization", "Bearer "+apiTokenDecrypted)
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
