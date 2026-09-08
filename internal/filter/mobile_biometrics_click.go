package filter

import "ad-event-processor/internal/domain"

func MobileGyroFlat(events []domain.BehaviorTelemetryEvent) bool {
	return summarizeMobileBiometrics(events).gyroFlat == 1
}

func TouchPressureMissing(mobileUA bool, events []domain.BehaviorTelemetryEvent) bool {
	if !mobileUA {
		return false
	}
	hasTouch := false
	for i := range events {
		e := events[i]
		if e.T != "touchstart" && e.T != "touchmove" {
			continue
		}
		hasTouch = true
		if e.Force > 0 || e.RadiusX > 0 || e.RadiusY > 0 {
			return false
		}
	}
	return hasTouch
}

func CampaignRequiresClickMobileBiometrics(camp *domain.Campaign) bool {
	if camp == nil {
		return false
	}
	return camp.SafePageEnabled && camp.AttestationEnabled && camp.MobileBiometricsClickEnabled
}
