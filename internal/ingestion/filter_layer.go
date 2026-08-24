package ingestion

import (
	"context"

	"ad-event-processor/internal/domain"

	redis "github.com/redis/go-redis/v9"
)

const fraudBlacklistKey = "blacklist:fraud"

type FraudLayer uint8

const (
	FraudLayerNone FraudLayer = iota
	FraudLayerL2Shadow
	FraudLayerL1Reject
)

func decideFraudLayer(acc *fraudAccumulator, tier FraudTier) FraudLayer {
	if acc == nil || acc.count == 0 {
		return FraudLayerNone
	}
	if acc.hasFlags(fraudSignalL3) {
		return FraudLayerL1Reject
	}
	if acc.countFlags(fraudSignalL1High) >= 2 {
		return FraudLayerL1Reject
	}
	if acc.countFlags(fraudSignalL1High) >= 1 ||
		acc.countFlags(fraudSignalL2Weak) >= 1 ||
		tier == FraudTierSuspect ||
		tier == FraudTierIVT ||
		tier == FraudTierBlock {
		return FraudLayerL2Shadow
	}
	return FraudLayerNone
}

func applyFraudLayerDecision(evt *domain.Event, acc *fraudAccumulator, camp *domain.Campaign, boost uint8) (FraudLayer, error) {
	if evt == nil {
		return FraudLayerNone, nil
	}
	evt.ShadowEvent = false

	if acc != nil && boost > 0 && !acc.boostApplied {
		sum := acc.score + uint32(boost)
		if sum > 100 {
			sum = 100
		}
		acc.score = sum
		acc.boostApplied = true
	}

	tier := applyFraudAccumulatorForCampaign(evt, acc, camp)
	if acc == nil || acc.count == 0 {
		return FraudLayerNone, nil
	}

	layer := decideFraudLayer(acc, tier)
	recordFraudMetrics(acc, tier, layer)

	switch layer {
	case FraudLayerL1Reject:
		return FraudLayerL1Reject, ErrFraudDetected
	case FraudLayerL2Shadow:
		evt.ShadowEvent = true
		return FraudLayerL2Shadow, nil
	default:
		return FraudLayerNone, nil
	}
}

type FraudBlacklistFilter struct {
	rdbs []redis.UniversalClient
}

func NewFraudBlacklistFilter(rdbs []redis.UniversalClient) *FraudBlacklistFilter {
	if len(rdbs) == 0 {
		return nil
	}
	return &FraudBlacklistFilter{rdbs: rdbs}
}

func (f *FraudBlacklistFilter) Check(ctx context.Context, evt *domain.Event) error {
	if f == nil || evt == nil || evt.IP == "" {
		return nil
	}
	rdb := pickLocalGlobalShard(f.rdbs)
	if rdb == nil {
		return nil
	}
	onList, err := rdb.SIsMember(ctx, fraudBlacklistKey, evt.IP).Result()
	if err != nil {
		return nil
	}
	if onList {
		addFraudSignal(evt, FraudReasonL3Blocklist)
	}
	return nil
}

func pickLocalGlobalShard(rdbs []redis.UniversalClient) redis.UniversalClient {
	if len(rdbs) == 0 {
		return nil
	}
	for i := 1; i < len(rdbs); i++ {
		if rdbs[i] != nil {
			return rdbs[i]
		}
	}
	return rdbs[0]
}
