package rtb

import (
	"sync/atomic"
)

type Registry struct {
	catalog               atomic.Pointer[catalogSnapshot]
	store                 *BudgetStore
	snapGen               atomic.Uint64
	clearingMode          atomic.Uint32
	targetingIndexEnabled atomic.Bool
	pendingCreatives      []CreativeData
	fcap                  fcapSnap
}

func NewRegistry(store *BudgetStore) *Registry {
	registry := &Registry{store: store}
	registry.clearingMode.Store(uint32(ClearingSecondPrice))
	empty := &catalogSnapshot{}
	for i := 0; i < geoShardCount; i++ {
		empty.shards[i] = &CampaignAuctionRegistry{}
	}
	registry.catalog.Store(empty)
	return registry
}

func (registry *Registry) SetClearingMode(mode ClearingMode) {
	registry.clearingMode.Store(uint32(mode))
}

func (registry *Registry) ClearingMode() ClearingMode {
	return ClearingMode(registry.clearingMode.Load())
}

func (registry *Registry) SetTargetingIndexEnabled(enabled bool) {
	registry.targetingIndexEnabled.Store(enabled)
}

func (registry *Registry) TargetingIndexEnabled() bool {
	return registry.targetingIndexEnabled.Load()
}

func (registry *Registry) LoadShard(idx uint32) *CampaignAuctionRegistry {
	snap := registry.catalog.Load()
	if snap == nil {
		return nil
	}
	return snap.shards[idx&geoShardMask]
}

func (registry *Registry) loadCatalog() *catalogSnapshot {
	return registry.catalog.Load()
}

func (registry *Registry) Store() *BudgetStore {
	return registry.store
}

func (registry *Registry) SetFcapSnapshot(snap *FcapSnapshot) {
	registry.fcap.store(snap)
}

func (registry *Registry) LoadFcapSnapshot() *FcapSnapshot {
	return registry.fcap.load()
}

func (registry *Registry) UpdateCreatives(creatives []CreativeData) {
	if len(creatives) == 0 {
		registry.pendingCreatives = nil
		return
	}
	registry.pendingCreatives = append(registry.pendingCreatives[:0], creatives...)
}

func (registry *Registry) UpdateCampaigns(campaigns []CampaignData) {
	var counts [geoShardCount]int
	for i := range campaigns {
		shardIdx := campaigns[i].GeoHashVal & geoShardMask
		counts[shardIdx]++
	}

	var registries [geoShardCount]*CampaignAuctionRegistry
	for shardIdx := 0; shardIdx < geoShardCount; shardIdx++ {
		n := counts[shardIdx]
		registries[shardIdx] = &CampaignAuctionRegistry{
			Count:                 n,
			CampaignIDs:           make([]CampaignID, n),
			Bids:                  make([]int64, n),
			CTRPPM:                make([]uint32, n),
			Reserves:              make([]int64, n),
			DailyBudgets:          make([]int64, n),
			PacingOpen:            make([]uint8, n),
			DeviceMasks:           make([]uint8, n),
			CategoryMasks:         make([]uint64, n),
			GeoHashes:             make([]uint32, n),
			Weights:               make([]uint32, n),
			BoostPPM:              make([]uint32, n),
			BudgetIndices:         make([]uint32, n),
			CustomerBudgetIndices: make([]uint32, n),
			DaypartMasks:          make([]uint32, n),
			TZOffsetSec:           make([]int32, n),
			ScheduleStart:         make([]int64, n),
			ScheduleEnd:           make([]int64, n),
			FreqLimits:            make([]uint32, n),
			FcapPrefixHash:        make([]uint64, n),
		}
	}

	var writeIndices [geoShardCount]int
	for i := range campaigns {
		c := &campaigns[i]
		shardIdx := c.GeoHashVal & geoShardMask
		reg := registries[shardIdx]
		wIdx := writeIndices[shardIdx]

		reg.CampaignIDs[wIdx] = c.ID
		reg.Bids[wIdx] = c.Bid
		reg.CTRPPM[wIdx] = normalizeCTRPPM(c.CTRPPM)
		reg.Reserves[wIdx] = c.Reserve
		reg.DailyBudgets[wIdx] = c.DailyBudget
		reg.PacingOpen[wIdx] = normalizePacingOpen(c.PacingOpen)
		reg.DeviceMasks[wIdx] = c.DeviceMask
		reg.CategoryMasks[wIdx] = c.CategoryMask
		reg.GeoHashes[wIdx] = c.GeoHashVal
		reg.Weights[wIdx] = c.Weight
		reg.BoostPPM[wIdx] = normalizeCTRPPM(c.BoostPPM)
		reg.BudgetIndices[wIdx] = registry.store.GetOrAllocateSlot(c.ID, c.Budget)
		reg.CustomerBudgetIndices[wIdx] = registry.store.GetOrAllocateCustomerSlot(c.CustomerID, c.CustomerBudget)
		reg.DaypartMasks[wIdx] = c.DaypartMask
		reg.TZOffsetSec[wIdx] = c.TZOffsetSec
		reg.ScheduleStart[wIdx] = c.ScheduleStart
		reg.ScheduleEnd[wIdx] = c.ScheduleEnd
		reg.FreqLimits[wIdx] = c.FreqLimit
		reg.FcapPrefixHash[wIdx] = c.FcapPrefixHash

		writeIndices[shardIdx]++
	}

	targetingEnabled := registry.targetingIndexEnabled.Load()
	shardCreatives := partitionCreativesByShard(campaigns, registry.pendingCreatives)
	for shardIdx := 0; shardIdx < geoShardCount; shardIdx++ {
		buildCreativeCache(registries[shardIdx], shardCreatives[shardIdx])
		buildGeoIndex(registries[shardIdx])
		if targetingEnabled {
			buildTargetingIndex(registries[shardIdx])
		}
		sortRegistryBuckets(registries[shardIdx])
	}

	registry.publishCatalog(registries)
}

func partitionCreativesByShard(campaigns []CampaignData, creatives []CreativeData) [geoShardCount][]CreativeData {
	var out [geoShardCount][]CreativeData
	if len(creatives) == 0 {
		return out
	}
	campaignShard := make(map[CampaignID]uint32, len(campaigns))
	for i := range campaigns {
		campaignShard[campaigns[i].ID] = campaigns[i].GeoHashVal & geoShardMask
	}
	for i := range creatives {
		shard, ok := campaignShard[creatives[i].CampaignID]
		if !ok {
			continue
		}
		out[shard] = append(out[shard], creatives[i])
	}
	return out
}

func (registry *Registry) publishCatalog(shards [geoShardCount]*CampaignAuctionRegistry) {
	registry.catalog.Store(&catalogSnapshot{shards: shards})
	registry.snapGen.Add(1)
}
