package licensing

import (
	"strconv"
	"time"

	"github.com/bidshard/ad-event-processor/internal/config"
)

type HeartbeatPolicy struct {
	OfflineGraceDays int
	RenewBeforeDays  int
}

func LoadHeartbeatPolicyFromEnv() HeartbeatPolicy {
	p := HeartbeatPolicy{
		OfflineGraceDays: 14,
		RenewBeforeDays:  7,
	}
	if v := config.LicenseEnv("OFFLINE_GRACE_DAYS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			p.OfflineGraceDays = n
		}
	}
	if v := config.LicenseEnv("RENEW_BEFORE_DAYS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			p.RenewBeforeDays = n
		}
	}
	if p.RenewBeforeDays > p.OfflineGraceDays {
		p.RenewBeforeDays = p.OfflineGraceDays
	}
	return p
}

func OfflineDays(offlineSince time.Time, now time.Time) int {
	if offlineSince.IsZero() {
		return 0
	}
	d := now.Sub(offlineSince)
	if d < 0 {
		return 0
	}
	return int(d / (24 * time.Hour))
}

func DetermineEffectiveState(claims *LicenseClaims, now time.Time, revoked bool, offlineSince time.Time, heartbeatOffline bool, policy HeartbeatPolicy) LicenseState {
	jwtState := DetermineState(claims, now, claims.Revoked)
	if jwtState == StateRevoked || jwtState == StateExpired {
		return jwtState
	}
	if jwtState == StateGrace {
		return StateGrace
	}
	if !heartbeatOffline || offlineSince.IsZero() {
		return StateActive
	}
	days := OfflineDays(offlineSince, now)
	if days >= policy.OfflineGraceDays {
		return StateExpired
	}
	tail := policy.RenewBeforeDays
	if tail <= 0 {
		return StateOfflineWarn
	}
	warnEnd := policy.OfflineGraceDays - tail
	if warnEnd < 0 {
		warnEnd = 0
	}
	if days >= warnEnd && policy.OfflineGraceDays > tail {
		return StateOfflineGrace
	}
	return StateOfflineWarn
}

func BannerSeverity(state LicenseState) string {
	switch state {
	case StateOfflineWarn:
		return "warn"
	case StateOfflineGrace:
		return "grace"
	default:
		return ""
	}
}

func IngestAllowed(state LicenseState) bool {
	switch state {
	case StateExpired, StateRevoked:
		return false
	default:
		return true
	}
}
