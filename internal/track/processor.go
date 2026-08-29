package track

import (
	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/filter"
)

type Status uint8

const (
	StatusAccepted Status = iota
	StatusFraudAccepted
	StatusRejected
	StatusInternalError
)

type Outcome struct {
	Status     Status
	RejectKind filter.FilterRejectKind
	LandingURL string
}

func FraudOutcome(registry domain.CampaignRegistry, evt *domain.Event) Outcome {
	if evt != nil && CampaignSilentRejectEnabled(registry, evt) {
		evt.SilentRejectEvent = true
		return Outcome{Status: StatusFraudAccepted, RejectKind: filter.FilterRejectFraud}
	}
	if evt != nil {
		evt.SilentRejectEvent = false
	}
	return Outcome{Status: StatusRejected, RejectKind: filter.FilterRejectFraudBlocked}
}

func CampaignSilentRejectEnabled(registry domain.CampaignRegistry, evt *domain.Event) bool {
	if registry == nil || evt == nil {
		return false
	}
	camp, ok := filter.GetCampaignFromEvent(registry, evt)
	if !ok || camp == nil {
		return false
	}
	return camp.SilentRejectEnabled
}
