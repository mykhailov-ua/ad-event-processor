package filter

import (
	"context"
	"time"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/metrics"
	"ad-event-processor/pkg/antifraudtelemetry"
)

type AntifraudTelemetryFilter struct {
	registry     domain.CampaignRegistry
	enabled      bool
	verifyCrypto bool
	secret       []byte
}

func NewAntifraudTelemetryFilter(registry domain.CampaignRegistry) *AntifraudTelemetryFilter {
	return &AntifraudTelemetryFilter{registry: registry}
}

func (f *AntifraudTelemetryFilter) SetEnabled(enabled bool) {
	if f != nil {
		f.enabled = enabled
	}
}

func (f *AntifraudTelemetryFilter) SetVerifyCrypto(enabled bool) {
	if f != nil {
		f.verifyCrypto = enabled
	}
}

func (f *AntifraudTelemetryFilter) SetChallengeSecret(secret []byte) {
	if f == nil {
		return
	}
	f.secret = secret
}

func (f *AntifraudTelemetryFilter) Check(ctx context.Context, evt *domain.Event) error {
	if f == nil || evt == nil || !f.enabled {
		return nil
	}
	if evt.Type != "conversion" && evt.Type != "click" {
		return nil
	}
	if ScanUAFamily(evt.UA) == UAFamilyUnknown {
		return nil
	}
	if f.registry == nil {
		return nil
	}
	camp, ok := f.registry.GetCampaign(evt.CampaignID)
	if !ok || camp == nil {
		return nil
	}
	if !campaignRequiresAntifraudTelemetry(camp) {
		return nil
	}
	if evt.AntifraudSet == 0 {
		metrics.AntifraudTelemetryMissingTotal.Inc()
		AddFraudSignal(evt, FraudReasonAntifraudTelemetryMissing)
		return nil
	}
	snap := evt.AntifraudSnapshot
	if f.verifyCrypto && len(f.secret) > 0 {
		verifyAntifraudCrypto(evt, snap, f.secret)
	}
	in := antifraudtelemetry.Input{
		NavStartPTMs:         snap.NavStartPTMs,
		DwellMs:              snap.DwellMs,
		FooterReachMs:        snap.FooterReachMs,
		TrustedRatioMilli:    snap.TrustedRatioMilli,
		PointerCVMilli:       snap.PointerCVMilli,
		PointerDtCVMilli:     snap.PointerDtCVMilli,
		ScrollCVMilli:        snap.ScrollCVMilli,
		ScrollJerkMilli:      snap.ScrollJerkMilli,
		TouchIntervalCVMilli: snap.TouchIntervalCVMilli,
		RafCvMilli:           snap.RafCvMilli,
		RuntimeLeak:          snap.RuntimeLeak,
		Webdriver:            snap.Webdriver,
		AutomationLeak:       snap.AutomationLeak,
		ServerRTTSynMS:       evt.RTTSynMS,
		ServerTTFBMS:         evt.TTFBAppMS,
		ConnTimingSet:        evt.ConnTimingSet,
	}
	if snap.RTTSampleCount > 0 {
		n := int(snap.RTTSampleCount)
		if n > domain.AntifraudMaxRTTSamples {
			n = domain.AntifraudMaxRTTSamples
		}
		in.RTTSamples = snap.RTTSamples[:n]
	}
	v := antifraudtelemetry.Score(in)
	if v.AutomationLeak {
		metrics.AntifraudAutomationLeakTotal.Inc()
		AddFraudSignal(evt, FraudReasonAntifraudAutomationLeak)
	}
	if v.UntrustedEvents {
		metrics.AntifraudUntrustedEventsTotal.Inc()
		AddFraudSignal(evt, FraudReasonAntifraudUntrustedEvents)
	}
	if v.TemplateBehavior {
		metrics.AntifraudBehaviorTemplateTotal.Inc()
		AddFraudSignal(evt, FraudReasonAntifraudBehaviorTemplate)
	}
	if v.FastProbe {
		metrics.AntifraudFastProbeTotal.Inc()
		AddFraudSignal(evt, FraudReasonAntifraudFastProbe)
	}
	if v.ProxyJitter {
		metrics.AntifraudNetworkJitterTotal.Inc()
		AddFraudSignal(evt, FraudReasonAntifraudNetworkJitter)
	}
	if v.EmptyKinematics {
		metrics.AntifraudEmptyKinematicsTotal.Inc()
		AddFraudSignal(evt, FraudReasonAntifraudEmptyKinematics)
	}
	if v.RttMissing {
		metrics.AntifraudRttMissingTotal.Inc()
		AddFraudSignal(evt, FraudReasonAntifraudRttMissing)
	}
	return nil
}

func verifyAntifraudCrypto(evt *domain.Event, snap domain.AntifraudSnapshot, secret []byte) {
	if snap.ChallengeTokenLen == 0 || snap.PoWNonce == 0 || snap.TelemetryMAC[0] == 0 {
		metrics.AntifraudSignatureInvalidTotal.Inc()
		AddFraudSignal(evt, FraudReasonAntifraudSignatureInvalid)
		return
	}
	token := antifraudChallengeTokenString(snap)
	now := time.Now().Unix()
	info, err := antifraudtelemetry.ParseChallengeToken(secret, token, evt.CampaignID, now)
	if err != nil {
		metrics.AntifraudSignatureInvalidTotal.Inc()
		AddFraudSignal(evt, FraudReasonAntifraudSignatureInvalid)
		return
	}
	if !antifraudtelemetry.VerifyPoW(info.Salt[:], snap.PoWNonce, info.Difficulty) {
		metrics.AntifraudPowInvalidTotal.Inc()
		AddFraudSignal(evt, FraudReasonAntifraudPowInvalid)
		return
	}
	if !antifraudtelemetry.VerifyTelemetryMACDerived(
		token,
		snap.PoWNonce,
		snap.DwellMs,
		snap.PointerCVMilli,
		snap.RafCvMilli,
		snap.RuntimeLeak,
		snap.AutomationLeak,
		snap.TelemetryMAC[:],
	) {
		metrics.AntifraudSignatureInvalidTotal.Inc()
		AddFraudSignal(evt, FraudReasonAntifraudSignatureInvalid)
	}
}

func campaignRequiresAntifraudTelemetry(camp *domain.Campaign) bool {
	return camp.SafePageEnabled && camp.AttestationEnabled
}

func antifraudChallengeTokenString(snap domain.AntifraudSnapshot) string {
	if snap.ChallengeTokenLen == 0 {
		return ""
	}
	return UnsafeString(snap.ChallengeToken[:snap.ChallengeTokenLen])
}
