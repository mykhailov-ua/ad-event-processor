package trafficoptimizer

import (
	"math/rand"

	"ad-event-processor/internal/flow"

	"github.com/google/uuid"
)

type FlowWeightApplier interface {
	ApplyThompson(
		raw []byte,
		campaignIDs []uuid.UUID,
		landerByCampaign map[uuid.UUID]map[uuid.UUID]flow.EntityBanditStat,
		offerByCampaign map[uuid.UUID]map[uuid.UUID]flow.EntityBanditStat,
		rng *rand.Rand,
		cfg flow.BanditApplyConfig,
	) ([]byte, bool, error)
	ApplyProportional(
		raw []byte,
		campaignIDs []uuid.UUID,
		landerByCampaign map[uuid.UUID]map[uuid.UUID]flow.EntityBanditStat,
		offerByCampaign map[uuid.UUID]map[uuid.UUID]flow.EntityBanditStat,
		cfg flow.BanditApplyConfig,
	) ([]byte, bool, error)
}

type flowBanditApplier struct{}

func NewFlowBanditApplier() FlowWeightApplier {
	return flowBanditApplier{}
}

func (flowBanditApplier) ApplyThompson(
	raw []byte,
	campaignIDs []uuid.UUID,
	landerByCampaign map[uuid.UUID]map[uuid.UUID]flow.EntityBanditStat,
	offerByCampaign map[uuid.UUID]map[uuid.UUID]flow.EntityBanditStat,
	rng *rand.Rand,
	cfg flow.BanditApplyConfig,
) ([]byte, bool, error) {
	return flow.ApplyFlowBanditThompson(raw, campaignIDs, landerByCampaign, offerByCampaign, rng, cfg)
}

func (flowBanditApplier) ApplyProportional(
	raw []byte,
	campaignIDs []uuid.UUID,
	landerByCampaign map[uuid.UUID]map[uuid.UUID]flow.EntityBanditStat,
	offerByCampaign map[uuid.UUID]map[uuid.UUID]flow.EntityBanditStat,
	cfg flow.BanditApplyConfig,
) ([]byte, bool, error) {
	return flow.ApplyFlowBanditProportional(raw, campaignIDs, landerByCampaign, offerByCampaign, cfg)
}
