package ingestion

import (
	"context"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/metrics"
)

type JSONSerializationFilter struct {
	registry domain.CampaignRegistry
	enabled  bool
}

func NewJSONSerializationFilter(registry domain.CampaignRegistry) *JSONSerializationFilter {
	return &JSONSerializationFilter{registry: registry}
}

func (f *JSONSerializationFilter) SetEnabled(enabled bool) {
	if f == nil {
		return
	}
	f.enabled = enabled
}

func (f *JSONSerializationFilter) Check(ctx context.Context, evt *domain.Event) error {
	if f == nil || !f.enabled || evt == nil || evt.JSONSerializationFlags == 0 {
		return nil
	}
	if f.registry == nil {
		return nil
	}
	camp, ok := f.registry.GetCampaign(evt.CampaignID)
	if !ok || camp == nil || !camp.JSONSerializationEnabled {
		return nil
	}
	metrics.JSONSerializationBotTotal.Inc()
	addFraudSignal(evt, FraudReasonJSONSerializationBot)
	return nil
}
