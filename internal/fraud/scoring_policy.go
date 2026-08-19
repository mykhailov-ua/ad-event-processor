package fraud

import "github.com/bidshard/ad-event-processor/internal/domain"

type TierDecision struct {
	Tier                FraudTier
	Score               int
	MLProbability       float64
	AdjustedProbability float64
	ResidentialProxy    bool
	StructuralFraud     bool
	FPGuardApplied      bool
}

type rowMetrics struct {
	events             float64
	clicks             float64
	ctr                float64
	spendRatio         float64
	uniqueUsers        float64
	uniqueUAs          float64
	eventsPerUser      float64
	impressionPressure float64
	userClickGap       float64
	eventsPerUA        float64
	clicksPerUser      float64
	usersPerUA         float64
}

func metricsFromRow(row FeatureRow) rowMetrics {
	events := float64(row.Events)
	clicks := float64(row.Clicks)
	users := float64(row.UniqueUsers)
	uas := float64(row.UniqueUAs)

	return rowMetrics{
		events:             events,
		clicks:             clicks,
		ctr:                safeRatio(clicks, events),
		spendRatio:         safeRatio(float64(row.SpendMicro), float64(row.BudgetLimitMicro)),
		uniqueUsers:        users,
		uniqueUAs:          uas,
		eventsPerUser:      safeRatio(events, users),
		impressionPressure: safeRatio(events, clicks+1),
		userClickGap:       safeRatio(users, clicks+1),
		eventsPerUA:        safeRatio(events, uas),
		clicksPerUser:      safeRatio(clicks, users),
		usersPerUA:         safeRatio(users, uas),
	}
}

func ResidentialProxySignalWithConfig(row FeatureRow, cfg PolicyConfig) bool {
	m := metricsFromRow(row)
	if m.events < cfg.ProxyMinEvents {
		return false
	}
	if m.ctr > cfg.ProxyMaxCTR {
		return false
	}
	if m.uniqueUsers < cfg.ProxyMinUsers {
		return false
	}
	if m.userClickGap < cfg.ProxyMinUserClickGap {
		return false
	}
	if m.eventsPerUser < cfg.ProxyMinEventsPerUser {
		return false
	}
	if m.impressionPressure < cfg.ProxyMinImpressionPressure {
		return false
	}
	if m.usersPerUA < cfg.ProxyMinUsersPerUA {
		return false
	}
	if m.clicks < cfg.ProxyMinClicks {
		return false
	}
	return true
}

func ResidentialProxySignal(row FeatureRow) bool {
	return ResidentialProxySignalWithConfig(row, GetPolicyConfig())
}

func StructuralFraudSignalWithConfig(row FeatureRow, cfg PolicyConfig) bool {
	m := metricsFromRow(row)
	if m.ctr > cfg.StructuralHighCTR && m.uniqueUsers <= cfg.StructuralMaxUsers {
		return true
	}
	if m.uniqueUAs <= 1 && m.events >= cfg.StructuralMinEvents {
		return true
	}
	if m.eventsPerUA >= cfg.StructuralMinEventsPerUA {
		return true
	}
	if m.clicksPerUser >= cfg.StructuralMinClicksPerUser {
		return true
	}
	if m.spendRatio > cfg.StructuralSpendRatio && m.ctr > cfg.StructuralSpendMinCTR {
		return true
	}
	return false
}

func StructuralFraudSignal(row FeatureRow) bool {
	return StructuralFraudSignalWithConfig(row, GetPolicyConfig())
}

func AdjustProbabilityWithConfig(row FeatureRow, mlProbability float64, cfg PolicyConfig) (float64, bool, bool, bool) {
	prob := mlProbability
	proxy := ResidentialProxySignalWithConfig(row, cfg)
	structural := StructuralFraudSignalWithConfig(row, cfg)
	fpGuard := false

	if proxy && prob < cfg.ResidentialProxyMaxML {
		if prob < cfg.ResidentialProxyFloor {
			prob = cfg.ResidentialProxyFloor
		}
	}

	blockProb := cfg.BlockProbability()
	if prob >= blockProb && !structural && !proxy {
		prob = cfg.FPGuardCap
		fpGuard = true
	}

	return prob, proxy, structural, fpGuard
}

func AdjustProbability(row FeatureRow, mlProbability float64) (probability float64, pass, suspect, block bool) {
	return AdjustProbabilityWithConfig(row, mlProbability, GetPolicyConfig())
}

func MapProbabilityTierWithThresholds(probability float64, pass, suspect, ivt, block uint8) (tier FraudTier, clamped int) {
	score := ProbabilityToFraudScore(probability)
	if pass == 0 {
		pass = domain.DefaultFraudThresholdPass
	}
	if suspect == 0 {
		suspect = domain.DefaultFraudThresholdSuspect
	}
	if ivt == 0 {
		ivt = domain.DefaultFraudThresholdIVT
	}
	if block == 0 {
		block = domain.DefaultFraudThresholdBlock
	}
	switch {
	case score <= int(pass):
		return FraudTierPass, score
	case score <= int(suspect):
		return FraudTierSuspect, score
	case score <= int(ivt):
		return FraudTierIVT, score
	case score <= int(block):
		return FraudTierBlock, score
	default:
		return FraudTierBlock, score
	}
}

func DecideWithPolicy(row FeatureRow, mlProbability float64, cfg PolicyConfig) TierDecision {
	adjusted, proxy, structural, fpGuard := AdjustProbabilityWithConfig(row, mlProbability, cfg)
	tier, score := MapProbabilityTierWithThresholds(adjusted, cfg.TierPass, cfg.TierSuspect, cfg.TierIVT, cfg.TierBlock)
	return TierDecision{
		Tier:                tier,
		Score:               score,
		MLProbability:       mlProbability,
		AdjustedProbability: adjusted,
		ResidentialProxy:    proxy,
		StructuralFraud:     structural,
		FPGuardApplied:      fpGuard,
	}
}

func DecideWithCampaign(row FeatureRow, mlProbability float64, pass, suspect, ivt, block uint8) TierDecision {
	cfg := GetPolicyConfig().WithCampaignThresholds(pass, suspect, ivt, block)
	return DecideWithPolicy(row, mlProbability, cfg)
}

func Decide(row FeatureRow, mlProbability float64) TierDecision {
	return DecideWithPolicy(row, mlProbability, GetPolicyConfig())
}

func ActionFraudPositive(decision TierDecision, blockOnly bool) bool {
	if blockOnly {
		return decision.Tier == FraudTierBlock
	}
	switch decision.Tier {
	case FraudTierSuspect, FraudTierIVT, FraudTierBlock:
		return true
	default:
		return false
	}
}
