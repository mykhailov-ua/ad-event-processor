package ingestion

import (
	"context"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/metrics"
)

type BehaviorTelemetryFilter struct {
	registry domain.CampaignRegistry
	enabled  bool
}

func NewBehaviorTelemetryFilter(registry domain.CampaignRegistry) *BehaviorTelemetryFilter {
	return &BehaviorTelemetryFilter{registry: registry}
}

func (f *BehaviorTelemetryFilter) SetEnabled(enabled bool) {
	if f == nil {
		return
	}
	f.enabled = enabled
}

func (f *BehaviorTelemetryFilter) Check(ctx context.Context, evt *domain.Event) error {
	if f == nil || !f.enabled || evt == nil || evt.Type != "conversion" {
		return nil
	}
	if scanUAFamily(evt.UA) == uaFamilyUnknown {
		return nil
	}
	if f.registry == nil {
		return nil
	}
	camp, ok := f.registry.GetCampaign(evt.CampaignID)
	if !ok || camp == nil || !campaignRequiresBehaviorTelemetry(camp) {
		return nil
	}
	if evt.TelemetrySet == 0 || len(evt.TelemetryEvents) == 0 {
		metrics.BehaviorTelemetryMissingTotal.Inc()
		addFraudSignal(evt, FraudReasonBehaviorTelemetryMissing)
		return nil
	}
	if checkBezierBot(behaviorTelemetryToVerifyEvents(evt.TelemetryEvents)) != "" {
		metrics.BehaviorBezierBotTotal.Inc()
		addFraudSignal(evt, FraudReasonBehaviorBezierBot)
	}
	return nil
}

func campaignRequiresBehaviorTelemetry(camp *domain.Campaign) bool {
	return camp.SafePageEnabled && camp.AttestationEnabled
}

func behaviorTelemetryToVerifyEvents(in []domain.BehaviorTelemetryEvent) []safePageVerifyEvent {
	if len(in) == 0 {
		return nil
	}
	out := make([]safePageVerifyEvent, len(in))
	for i := range in {
		out[i] = safePageVerifyEvent{
			T:  in[i].T,
			TS: in[i].TS,
			X:  in[i].X,
			Y:  in[i].Y,
		}
	}
	return out
}
