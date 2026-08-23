package licensing_test

import (
	"testing"
	"time"

	"github.com/bidshard/ad-event-processor/internal/licensing"
	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"
)

func TestProperty_P_C3_01_StatePure(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		claims := genLicenseClaims(t)
		now := genTimeNearClaims(t, claims)
		revoked := rapid.Bool().Draw(t, "revoked")

		s1 := licensing.DetermineState(claims, now, revoked)
		s2 := licensing.DetermineState(claims, now, revoked)
		require.Equal(t, s1, s2)
	})
}

func TestProperty_P_C3_02_JwtStateMonotonic(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		claims := genLicenseClaims(t)
		graceDays := claims.GraceDays
		if graceDays < 0 {
			graceDays = 0
		}

		tActive := claims.ValidFrom.Add(time.Hour)
		tGrace := claims.ValidUntil.Add(time.Hour)
		tExpired := claims.ValidUntil.Add(time.Duration(graceDays)*24*time.Hour + time.Hour)

		sActive := licensing.DetermineState(claims, tActive, false)
		sGrace := licensing.DetermineState(claims, tGrace, false)
		sExpired := licensing.DetermineState(claims, tExpired, false)

		require.LessOrEqual(t, jwtStateRank(sGrace), jwtStateRank(sActive))
		require.LessOrEqual(t, jwtStateRank(sExpired), jwtStateRank(sGrace))
	})
}

func TestProperty_P_C3_03_IngestAllowedStates(t *testing.T) {
	allowed := map[licensing.LicenseState]bool{
		licensing.StateActive:       true,
		licensing.StateOfflineWarn:  true,
		licensing.StateOfflineGrace: true,
		licensing.StateGrace:        true,
		licensing.StateExpired:      false,
		licensing.StateRevoked:      false,
	}
	for state, want := range allowed {
		require.Equal(t, want, licensing.IngestAllowed(state), "state=%s", state)
	}
}

func TestProperty_P_C4_01_EffectiveIdempotent(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		ent := genEntitlements(t)
		once := licensing.Effective(ent, ent)
		twice := licensing.Effective(once, once)
		require.True(t, entitlementsEqual(once, twice))
	})
}

func TestProperty_P_C4_02_EffectiveAssociative(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		a := genEntitlements(t)
		b := genEntitlements(t)
		c := genEntitlements(t)

		left := licensing.Effective(licensing.Effective(a, b), c)
		right := licensing.Effective(a, licensing.Effective(b, c))
		require.True(t, entitlementsEqual(left, right))
	})
}

func TestProperty_P_C4_03_DeploymentCeiling(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		dep := genEntitlements(t)
		cust := genEntitlements(t)
		eff := licensing.Effective(dep, cust)

		requireLimitCeiling(dep.Limits.MaxRPS, eff.Limits.MaxRPS)
		requireLimitCeiling(dep.Limits.MaxRequestsPerDay, eff.Limits.MaxRequestsPerDay)
		requireLimitCeiling(dep.Limits.MaxActiveCampaigns, eff.Limits.MaxActiveCampaigns)
		requireLimitCeiling(dep.Limits.MaxRegions, eff.Limits.MaxRegions)
		requireLimitCeiling(dep.Limits.MaxTenants, eff.Limits.MaxTenants)
		requireLimitCeiling(dep.Limits.MaxEventsPerMonth, eff.Limits.MaxEventsPerMonth)
		requireLimitCeiling(dep.Limits.MaxAPIKeys, eff.Limits.MaxAPIKeys)
		requireLimitCeiling(dep.Limits.MaxExportChunkBytes, eff.Limits.MaxExportChunkBytes)

		depFeat := dep.Features.Normalized()
		effFeat := eff.Features.Normalized()
		require.False(t, effFeat.RtbLive && !depFeat.RtbLive)
		require.False(t, effFeat.OpenRTBEnabled() && !depFeat.OpenRTBEnabled())
	})
}

func jwtStateRank(state licensing.LicenseState) int {
	switch state {
	case licensing.StateActive:
		return 3
	case licensing.StateGrace:
		return 2
	case licensing.StateExpired, licensing.StateRevoked:
		return 1
	default:
		return 0
	}
}

func genLicenseClaims(t *rapid.T) *licensing.LicenseClaims {
	validFrom := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC).Add(
		time.Duration(rapid.IntRange(0, 365).Draw(t, "valid_from_offset_days")) * 24 * time.Hour,
	)
	validDays := rapid.IntRange(1, 120).Draw(t, "valid_days")
	graceDays := rapid.IntRange(0, 30).Draw(t, "grace_days")
	return &licensing.LicenseClaims{
		ValidFrom:  validFrom,
		ValidUntil: validFrom.Add(time.Duration(validDays) * 24 * time.Hour),
		GraceDays:  graceDays,
	}
}

func genTimeNearClaims(t *rapid.T, claims *licensing.LicenseClaims) time.Time {
	offsetHours := rapid.IntRange(-48, 48).Draw(t, "now_offset_hours")
	return claims.ValidFrom.Add(time.Duration(offsetHours) * time.Hour)
}

func genEntitlements(t *rapid.T) licensing.Entitlements {
	return licensing.Entitlements{
		Limits: licensing.Limits{
			MaxRPS:              genLimit(t, "max_rps"),
			MaxRequestsPerDay:   genLimit(t, "max_requests_per_day"),
			MaxActiveCampaigns:  genLimit(t, "max_active_campaigns"),
			MaxRegions:          genLimit(t, "max_regions"),
			MaxTenants:          genLimit(t, "max_tenants"),
			MaxEventsPerMonth:   genLimit(t, "max_events_per_month"),
			MaxAPIKeys:          genLimit(t, "max_api_keys"),
			MaxExportChunkBytes: genLimit(t, "max_export_chunk_bytes"),
		},
		Features: licensing.FeatureSet{
			RtbLive:                  rapid.Bool().Draw(t, "rtb_live"),
			OpenRTBEngine:            rapid.Bool().Draw(t, "openrtb_engine"),
			IvtMLDetector:            rapid.Bool().Draw(t, "ivt_ml_detector"),
			EbpfXDPEdge:              rapid.Bool().Draw(t, "ebpf_xdp_edge"),
			MlFraudBoost:             rapid.Bool().Draw(t, "ml_fraud_boost"),
			MultiRegion:              rapid.Bool().Draw(t, "multi_region"),
			SlotMigration:            rapid.Bool().Draw(t, "slot_migration"),
			MarginGuard:              rapid.Bool().Draw(t, "margin_guard"),
			ExternalResidentialIntel: rapid.Bool().Draw(t, "external_residential_intel"),
		},
	}
}

func genLimit(t *rapid.T, name string) uint64 {
	if rapid.Bool().Draw(t, name+"_zero") {
		return 0
	}
	return uint64(rapid.IntRange(1, 1_000_000).Draw(t, name))
}

func requireLimitCeiling(depLimit, effLimit uint64) {
	if depLimit == 0 {
		return
	}
	if effLimit > depLimit {
		panic("effective limit exceeds deployment ceiling")
	}
}

func entitlementsEqual(a, b licensing.Entitlements) bool {
	af := a.Features.Normalized()
	bf := b.Features.Normalized()
	aLimits := a.Limits
	bLimits := b.Limits
	if aLimits.QuotaResetTimezone == "" {
		aLimits.QuotaResetTimezone = "UTC"
	}
	if bLimits.QuotaResetTimezone == "" {
		bLimits.QuotaResetTimezone = "UTC"
	}
	return aLimits == bLimits &&
		af.RtbLive == bf.RtbLive &&
		af.OpenRTBEngine == bf.OpenRTBEngine &&
		af.IvtMLDetector == bf.IvtMLDetector &&
		af.EbpfXDPEdge == bf.EbpfXDPEdge &&
		af.MlFraudBoost == bf.MlFraudBoost &&
		af.MultiRegion == bf.MultiRegion &&
		af.SlotMigration == bf.SlotMigration &&
		af.MarginGuard == bf.MarginGuard &&
		af.ExternalResidentialIntel == bf.ExternalResidentialIntel &&
		a.VolumeBand == b.VolumeBand
}
