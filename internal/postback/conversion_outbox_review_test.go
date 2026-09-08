package postback

import (
	"context"
	"testing"
	"time"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/metrics"

	"github.com/google/uuid"
	dto "github.com/prometheus/client_model/go"
)

func TestConversionPostbackEnqueuer_skipsReviewRoutedEvent(t *testing.T) {
	campID := uuid.New()
	store := &benchPostbackQuerier{hasConfig: true}
	enq := NewConversionPostbackEnqueuer(store)
	evt := &domain.Event{
		Type:              "conversion",
		CampaignID:        campID,
		ClickID:           "clk-review",
		ReviewRoutedEvent: true,
		Payload:           []byte(`{"goal_name":"lead"}`),
	}
	enq.OnBatchStored(context.Background(), []*domain.Event{evt})
	if store.outboxCalls != 0 {
		t.Fatalf("outbox calls %d", store.outboxCalls)
	}
}

func TestConversionPostbackEnqueuer_skipsReviewRoutedClick(t *testing.T) {
	campID := uuid.New()
	store := &benchPostbackQuerier{hasConfig: true}
	enq := NewConversionPostbackEnqueuer(store)
	enq.SetClickStore(&stubConversionClickStore{
		clicks: map[string]clickSnapshot{
			"clk-sandbox": {
				clickID:      "clk-sandbox",
				campaignID:   campID,
				createdAt:    time.Now().Add(-10 * time.Second),
				reviewRouted: true,
			},
		},
	})
	evt := &domain.Event{
		Type:       "conversion",
		CampaignID: campID,
		ClickID:    "clk-sandbox",
		Payload:    []byte(`{"goal_name":"lead"}`),
	}
	enq.OnBatchStored(context.Background(), []*domain.Event{evt})
	if store.outboxCalls != 0 {
		t.Fatalf("outbox calls %d", store.outboxCalls)
	}
}

func TestConversionPostbackEnqueuer_countsBrowserMissingEventID(t *testing.T) {
	before := readCounterValue(metrics.ConversionBrowserMissingTotal)
	campID := benchCampaignID
	store := &benchPostbackQuerier{hasConfig: true}
	enq := NewConversionPostbackEnqueuer(store)
	evt := &domain.Event{
		Type:       "conversion",
		CampaignID: campID,
		ClickID:    "clk-no-event-id",
		Payload:    []byte(`{"goal_name":"lead","fbclid":"fb-1"}`),
	}
	enq.OnBatchStored(context.Background(), []*domain.Event{evt})
	if store.outboxCalls != 1 {
		t.Fatalf("outbox calls %d", store.outboxCalls)
	}
	after := readCounterValue(metrics.ConversionBrowserMissingTotal)
	if after != before+1 {
		t.Fatalf("browser missing counter before=%f after=%f", before, after)
	}
}

func readCounterValue(c interface{ Write(*dto.Metric) error }) float64 {
	var m dto.Metric
	if err := c.Write(&m); err != nil {
		return 0
	}
	if m.Counter == nil {
		return 0
	}
	return m.Counter.GetValue()
}
