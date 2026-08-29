package filter

import (
	"errors"
	"sync/atomic"
	"unsafe"

	"ad-event-processor/internal/database"
	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/filter/netintel"

	"github.com/google/uuid"
)

const HexChars = "0123456789abcdef"

func AppendUUID(dst []byte, u uuid.UUID) []byte {
	for i := range len(u) {
		dst = append(dst, "0123456789abcdef"[u[i]>>4], "0123456789abcdef"[u[i]&0x0f])
		if i == 3 || i == 5 || i == 7 || i == 9 {
			dst = append(dst, '-')
		}
	}
	return dst
}

func ParseUUID(b []byte, dst *uuid.UUID) bool {
	if len(b) != 36 {
		return false
	}
	u, err := uuid.ParseBytes(b)
	if err != nil {
		return false
	}
	*dst = u
	return true
}

func UnsafeString(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	return unsafe.String(unsafe.SliceData(b), len(b))
}

func UnsafeBytes(s string) []byte {
	if len(s) == 0 {
		return nil
	}
	return unsafe.Slice(unsafe.StringData(s), len(s))
}

var registryQuantaFlush atomic.Pointer[func(uuid.UUID)]

func SetRegistryQuantaFlushHook(fn func(uuid.UUID)) {
	if fn == nil {
		registryQuantaFlush.Store(nil)
		return
	}
	registryQuantaFlush.Store(&fn)
}

func InvokeRegistryQuantaFlush(id uuid.UUID) {
	p := registryQuantaFlush.Load()
	if p == nil || *p == nil {
		return
	}
	(*p)(id)
}

var (
	ErrRateLimitExceeded      = errors.New("rate limit exceeded")
	ErrDuplicateEvent         = errors.New("duplicate event detected")
	ErrBudgetExhausted        = errors.New("budget exhausted")
	ErrCampaignNotFound       = errors.New("campaign not found in registry")
	ErrPacingExhausted        = errors.New("pacing exhausted")
	ErrFreqLimitExceeded      = domain.ErrFreqLimitExceeded
	ErrGeoBlocked             = errors.New("geo-targeting blocked")
	ErrScheduleBlocked        = errors.New("outside delivery schedule")
	ErrFraudDetected          = errors.New("fraud detected")
	ErrEmergencyBreakerActive = domain.ErrEmergencyBreakerActive
	ErrBidFloorNotMet         = errors.New("bid floor not met")
	ErrMigrationFenced        = domain.ErrMigrationFenced
	ErrLicenseExpired         = errors.New("license expired")
	ErrDailyQuotaExceeded     = errors.New("daily quota exceeded")
	ErrRegistryStale          = errors.New("registry stale: campaign unknown while control plane unreachable")
	ErrShardUnavailable       = errors.New("shard unavailable")
	ErrInfraNetwork           = errors.New("infrastructure network error")
	ErrFilterTimeout          = errors.New("filter timeout")
	ErrConsentDenied          = errors.New("consent not granted")
	ErrPlacementBlocked       = errors.New("placement blocked")
	ErrSegmentExcluded        = errors.New("segment excluded")
	ErrSegmentNotIncluded     = errors.New("segment not included")
)

type ASNLookup interface {
	LookupASN(ip string) (uint32, bool)
}

type FraudReasonID uint8

const (
	FraudReasonCodeDatacenterIP             = "datacenter_ip"
	FraudReasonCodeLowTTC                   = "low_ttc"
	FraudReasonCodeMissingImpTS             = "missing_imp_ts"
	FraudReasonCodeL3Blocklist              = "l3_blocklist"
	FraudReasonCodeTLSBlocklist             = "tls_blocklist"
	FraudReasonCodeDeviceMismatch           = "device_mismatch"
	FraudReasonCodeTCPMSSAnomaly            = "tcp_mss_anomaly"
	FraudReasonCodeTCPTunnelMSS             = "tcp_tunnel_mss"
	FraudReasonCodeTCPSynOSMismatch         = "tcp_syn_os_mismatch"
	FraudReasonCodeJSONSerializationBot     = "json_serialization_bot"
	FraudReasonCodeOSFingerprint            = "os_fingerprint_mismatch"
	FraudReasonCodeIPv4Rotation             = "ipv4_rotation"
	FraudReasonCodeResidentialProxy         = "residential_proxy"
	FraudReasonCodeAttestationMissing       = "attestation_missing"
	FraudReasonCodeModeratorIP              = "moderator_ip"
	FraudReasonCodeSecFetchAnomaly          = "sec_fetch_anomaly"
	FraudReasonCodeClientHintsMismatch      = "client_hints_mismatch"
	FraudReasonCodeTLSALPNMismatch          = "tls_alpn_mismatch"
	FraudReasonCodeH2SettingsMismatch       = "h2_settings_mismatch"
	FraudReasonCodeH2PseudoOrder            = "h2_pseudo_order_mismatch"
	FraudReasonCodeH2DowngradeArtifact      = "h2_downgrade_artifact"
	FraudReasonCodeHeaderOrderMismatch      = "header_order_mismatch"
	FraudReasonCodeAcceptEncodingMismatch   = "accept_encoding_mismatch"
	FraudReasonCodeAcceptLangGeoMismatch    = "accept_lang_geo_mismatch"
	FraudReasonCodeTLSJA4Mismatch           = "tls_ja4_mismatch"
	FraudReasonCodeBehaviorTelemetryMissing = "behavior_telemetry_missing"
	FraudReasonCodeBehaviorBezierBot        = "behavior_bezier_bot"
)

const (
	FraudReasonNone FraudReasonID = iota
	FraudReasonDatacenterIP
	FraudReasonLowTTC
	FraudReasonMissingImpTS
	FraudReasonL3Blocklist
	FraudReasonTLSBlocklist
	FraudReasonDeviceMismatch
	FraudReasonTCPMSSAnomaly
	FraudReasonTCPTunnelMSS
	FraudReasonTCPSynOSMismatch
	FraudReasonJSONSerializationBot
	FraudReasonOSFingerprint
	FraudReasonIPv4Rotation
	FraudReasonResidentialProxy
	FraudReasonAttestationMissing
	FraudReasonModeratorIP
	FraudReasonSecFetchAnomaly
	FraudReasonClientHintsMismatch
	FraudReasonTLSALPNMismatch
	FraudReasonH2SettingsMismatch
	FraudReasonH2PseudoOrder
	FraudReasonH2DowngradeArtifact
	FraudReasonHeaderOrderMismatch
	FraudReasonAcceptEncodingMismatch
	FraudReasonAcceptLangGeoMismatch
	FraudReasonTLSJA4Mismatch
	FraudReasonBehaviorTelemetryMissing
	FraudReasonBehaviorBezierBot
	fraudReasonCount
)

const FraudReasonCount = int(fraudReasonCount)

const (
	FraudSignalL1High uint8 = 1 << 0
	FraudSignalL2Weak uint8 = 1 << 1
	FraudSignalL3     uint8 = 1 << 2
)

type fraudReasonEntry struct {
	code   string
	weight uint8
	flags  uint8
}

var fraudReasonRegistry = [fraudReasonCount]fraudReasonEntry{
	FraudReasonNone:                     {},
	FraudReasonDatacenterIP:             {code: FraudReasonCodeDatacenterIP, weight: 45, flags: FraudSignalL1High},
	FraudReasonLowTTC:                   {code: FraudReasonCodeLowTTC, weight: 45, flags: FraudSignalL1High},
	FraudReasonMissingImpTS:             {code: FraudReasonCodeMissingImpTS, weight: 35, flags: FraudSignalL2Weak},
	FraudReasonL3Blocklist:              {code: FraudReasonCodeL3Blocklist, weight: 100, flags: FraudSignalL3},
	FraudReasonTLSBlocklist:             {code: FraudReasonCodeTLSBlocklist, weight: 45, flags: FraudSignalL1High},
	FraudReasonDeviceMismatch:           {code: FraudReasonCodeDeviceMismatch, weight: 35, flags: FraudSignalL2Weak},
	FraudReasonTCPMSSAnomaly:            {code: FraudReasonCodeTCPMSSAnomaly, weight: 35, flags: FraudSignalL2Weak},
	FraudReasonTCPTunnelMSS:             {code: FraudReasonCodeTCPTunnelMSS, weight: 35, flags: FraudSignalL2Weak},
	FraudReasonTCPSynOSMismatch:         {code: FraudReasonCodeTCPSynOSMismatch, weight: 35, flags: FraudSignalL2Weak},
	FraudReasonJSONSerializationBot:     {code: FraudReasonCodeJSONSerializationBot, weight: 35, flags: FraudSignalL2Weak},
	FraudReasonOSFingerprint:            {code: FraudReasonCodeOSFingerprint, weight: 35, flags: FraudSignalL2Weak},
	FraudReasonIPv4Rotation:             {code: FraudReasonCodeIPv4Rotation, weight: 35, flags: FraudSignalL2Weak},
	FraudReasonResidentialProxy:         {code: FraudReasonCodeResidentialProxy, weight: 35, flags: FraudSignalL2Weak},
	FraudReasonAttestationMissing:       {code: FraudReasonCodeAttestationMissing, weight: 35, flags: FraudSignalL2Weak},
	FraudReasonModeratorIP:              {code: FraudReasonCodeModeratorIP, weight: 45, flags: FraudSignalL1High},
	FraudReasonSecFetchAnomaly:          {code: FraudReasonCodeSecFetchAnomaly, weight: 35, flags: FraudSignalL2Weak},
	FraudReasonClientHintsMismatch:      {code: FraudReasonCodeClientHintsMismatch, weight: 35, flags: FraudSignalL2Weak},
	FraudReasonTLSALPNMismatch:          {code: FraudReasonCodeTLSALPNMismatch, weight: 35, flags: FraudSignalL2Weak},
	FraudReasonH2SettingsMismatch:       {code: FraudReasonCodeH2SettingsMismatch, weight: 35, flags: FraudSignalL2Weak},
	FraudReasonH2PseudoOrder:            {code: FraudReasonCodeH2PseudoOrder, weight: 35, flags: FraudSignalL2Weak},
	FraudReasonH2DowngradeArtifact:      {code: FraudReasonCodeH2DowngradeArtifact, weight: 35, flags: FraudSignalL2Weak},
	FraudReasonHeaderOrderMismatch:      {code: FraudReasonCodeHeaderOrderMismatch, weight: 35, flags: FraudSignalL2Weak},
	FraudReasonAcceptEncodingMismatch:   {code: FraudReasonCodeAcceptEncodingMismatch, weight: 35, flags: FraudSignalL2Weak},
	FraudReasonAcceptLangGeoMismatch:    {code: FraudReasonCodeAcceptLangGeoMismatch, weight: 35, flags: FraudSignalL2Weak},
	FraudReasonTLSJA4Mismatch:           {code: FraudReasonCodeTLSJA4Mismatch, weight: 35, flags: FraudSignalL2Weak},
	FraudReasonBehaviorTelemetryMissing: {code: FraudReasonCodeBehaviorTelemetryMissing, weight: 35, flags: FraudSignalL2Weak},
	FraudReasonBehaviorBezierBot:        {code: FraudReasonCodeBehaviorBezierBot, weight: 35, flags: FraudSignalL2Weak},
}

func FraudReasonCode(id FraudReasonID) string {
	if id >= fraudReasonCount {
		return ""
	}
	return fraudReasonRegistry[id].code
}

func FraudSignalWeight(id FraudReasonID) uint8 {
	if id >= fraudReasonCount {
		return 0
	}
	return fraudReasonRegistry[id].weight
}

func FraudSignalFlags(id FraudReasonID) uint8 {
	if id >= fraudReasonCount {
		return 0
	}
	return fraudReasonRegistry[id].flags
}

type FilterRejectKind uint8

const (
	FilterRejectEmergencyBreaker FilterRejectKind = iota
	FilterRejectRateLimit
	FilterRejectDuplicate
	FilterRejectBudget
	FilterRejectPacing
	FilterRejectFreq
	FilterRejectGeo
	FilterRejectSchedule
	FilterRejectCampaignNotFound
	FilterRejectBidFloor
	FilterRejectTimeout
	FilterRejectFraud
	FilterRejectConsent
	FilterRejectInfra
	FilterRejectLicenseExpired
	FilterRejectDailyQuotaExceeded
	FilterRejectPlacementBlocked
	FilterRejectSegmentExcluded
	FilterRejectSegmentNotIncluded
	FilterRejectRegistryStale
	FilterRejectShardUnavailable
	FilterRejectProducerOverload
	FilterRejectFraudBlocked
)

func ClassifyFilterErr(err error) (FilterRejectKind, bool) {
	switch {
	case errors.Is(err, ErrEmergencyBreakerActive):
		return FilterRejectEmergencyBreaker, true
	case errors.Is(err, ErrFilterTimeout):
		return FilterRejectTimeout, true
	case errors.Is(err, ErrRateLimitExceeded):
		return FilterRejectRateLimit, true
	case errors.Is(err, ErrDuplicateEvent):
		return FilterRejectDuplicate, true
	case errors.Is(err, ErrBudgetExhausted):
		return FilterRejectBudget, true
	case errors.Is(err, ErrPacingExhausted):
		return FilterRejectPacing, true
	case errors.Is(err, ErrFreqLimitExceeded):
		return FilterRejectFreq, true
	case errors.Is(err, ErrGeoBlocked):
		return FilterRejectGeo, true
	case errors.Is(err, ErrScheduleBlocked):
		return FilterRejectSchedule, true
	case errors.Is(err, ErrCampaignNotFound):
		return FilterRejectCampaignNotFound, true
	case errors.Is(err, ErrRegistryStale):
		return FilterRejectRegistryStale, true
	case errors.Is(err, ErrShardUnavailable):
		return FilterRejectShardUnavailable, true
	case errors.Is(err, ErrBidFloorNotMet):
		return FilterRejectBidFloor, true
	case errors.Is(err, ErrMigrationFenced):
		return FilterRejectInfra, true
	case errors.Is(err, ErrFraudDetected):
		return FilterRejectFraud, true
	case errors.Is(err, ErrConsentDenied):
		return FilterRejectConsent, true
	case errors.Is(err, ErrLicenseExpired):
		return FilterRejectLicenseExpired, true
	case errors.Is(err, ErrDailyQuotaExceeded):
		return FilterRejectDailyQuotaExceeded, true
	case errors.Is(err, ErrPlacementBlocked):
		return FilterRejectPlacementBlocked, true
	case errors.Is(err, ErrSegmentExcluded):
		return FilterRejectSegmentExcluded, true
	case errors.Is(err, ErrSegmentNotIncluded):
		return FilterRejectSegmentNotIncluded, true
	case isInfraFilterErr(err):
		return FilterRejectInfra, true
	default:
		return 0, false
	}
}

func isInfraFilterErr(err error) bool {
	if errors.Is(err, database.ErrRedisCircuitOpen) || errors.Is(err, ErrInfraNetwork) {
		return true
	}
	return database.IsNetworkOrSystemError(err)
}

func BytesEqualFoldASCII(b []byte, lit string) bool {
	if len(b) != len(lit) {
		return false
	}
	for i := range len(lit) {
		c := b[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		if c != lit[i] {
			return false
		}
	}
	return true
}

func UAClaimsChromeNotChromium(ua string) bool {
	return netintel.UAClaimsChromeNotChromium(ua)
}
