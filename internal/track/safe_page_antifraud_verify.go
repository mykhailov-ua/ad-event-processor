package track

import (
	"encoding/json"
	"time"

	"ad-event-processor/internal/domain"
	"ad-event-processor/pkg/antifraudtelemetry"

	"github.com/google/uuid"
)

const (
	safePageAttestAntifraudMissing    = "antifraud_missing"
	safePageAttestAntifraudInvalid    = "antifraud_signature_invalid"
	safePageAttestAntifraudPowInvalid = "antifraud_pow_invalid"
)

func RequiresSafePageAntifraudCrypto(camp *domain.Campaign, antifraudTelemetryEnabled bool) bool {
	if camp == nil || !antifraudTelemetryEnabled {
		return false
	}
	if !camp.SafePageEnabled || !camp.AttestationEnabled {
		return false
	}
	return !camp.AttestationMode.UsesTelemetryStealthBundle()
}

type antifraudVerifyJSON struct {
	ChallengeToken string `json:"challenge_token"`
	PoWNonce       uint32 `json:"pow_nonce"`
	TelemetryMAC   string `json:"telemetry_mac"`
	DwellMs        uint32 `json:"dwell_ms"`
	PointerCVMilli uint16 `json:"pointer_cv_milli"`
	RafCvMilli     uint16 `json:"raf_cv_milli"`
	RuntimeLeak    uint8  `json:"runtime_leak"`
	AutomationLeak uint8  `json:"automation_leak"`
}

func ParseAntifraudSnapshotFromJSON(raw []byte) (domain.AntifraudSnapshot, bool) {
	if len(raw) == 0 {
		return domain.AntifraudSnapshot{}, false
	}
	var in antifraudVerifyJSON
	if err := json.Unmarshal(raw, &in); err != nil {
		return domain.AntifraudSnapshot{}, false
	}
	if in.ChallengeToken == "" || in.PoWNonce == 0 || len(in.TelemetryMAC) != 32 {
		return domain.AntifraudSnapshot{}, false
	}
	if len(in.ChallengeToken) > domain.AntifraudMaxChallengeToken {
		return domain.AntifraudSnapshot{}, false
	}
	var snap domain.AntifraudSnapshot
	snap.ChallengeTokenLen = uint8(len(in.ChallengeToken))
	copy(snap.ChallengeToken[:], in.ChallengeToken)
	copy(snap.TelemetryMAC[:], in.TelemetryMAC)
	snap.PoWNonce = in.PoWNonce
	snap.DwellMs = in.DwellMs
	snap.PointerCVMilli = in.PointerCVMilli
	snap.RafCvMilli = in.RafCvMilli
	snap.RuntimeLeak = in.RuntimeLeak
	snap.AutomationLeak = in.AutomationLeak
	return snap, true
}

func EvaluateSafePageAntifraudCrypto(campaignID uuid.UUID, snap domain.AntifraudSnapshot, secret []byte, nowUnix int64) (bool, string) {
	if len(secret) == 0 {
		return false, ""
	}
	if snap.ChallengeTokenLen == 0 || snap.PoWNonce == 0 || snap.TelemetryMAC[0] == 0 {
		return true, safePageAttestAntifraudInvalid
	}
	token := string(snap.ChallengeToken[:snap.ChallengeTokenLen])
	if nowUnix == 0 {
		nowUnix = time.Now().Unix()
	}
	info, err := antifraudtelemetry.ParseChallengeToken(secret, token, campaignID, nowUnix)
	if err != nil {
		return true, safePageAttestAntifraudInvalid
	}
	if !antifraudtelemetry.VerifyPoW(info.Salt[:], snap.PoWNonce, info.Difficulty) {
		return true, safePageAttestAntifraudPowInvalid
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
		return true, safePageAttestAntifraudInvalid
	}
	return false, ""
}
