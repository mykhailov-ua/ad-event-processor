package filter

import (
	"ad-event-processor/internal/domain"
)

func clickTierMinForFilter(f EventFilter) domain.ClickFilterTier {
	switch f {
	case nil:
		return domain.ClickFilterTierFull
	default:
		switch f.(type) {
		case *LicenseFilter, *LicenseRPSFilter, *LicenseMonthlyEventsFilter:
			return domain.ClickFilterTierRedirectOnly
		case *EmergencyBreakerFilter, *GeoFilter, *ScheduleFilter:
			return domain.ClickFilterTierLight
		default:
			return domain.ClickFilterTierFull
		}
	}
}

func filterAllowedForClickTier(f EventFilter, tier domain.ClickFilterTier) bool {
	if tier == "" || tier == domain.ClickFilterTierFull {
		return true
	}
	minTier := clickTierMinForFilter(f)
	switch tier {
	case domain.ClickFilterTierRedirectOnly:
		return minTier == domain.ClickFilterTierRedirectOnly
	case domain.ClickFilterTierLight:
		return minTier == domain.ClickFilterTierRedirectOnly || minTier == domain.ClickFilterTierLight
	default:
		return true
	}
}

// FilterAllowedForClickTier exports tier gating for ingest holdout tests.
func FilterAllowedForClickTier(f EventFilter, tier domain.ClickFilterTier) bool {
	return filterAllowedForClickTier(f, tier)
}

// ClickTierSkipsUnifiedFilter reports whether unified budget debit is skipped for tier.
func ClickTierSkipsUnifiedFilter(tier domain.ClickFilterTier) bool {
	return tier == domain.ClickFilterTierLight || tier == domain.ClickFilterTierRedirectOnly
}
