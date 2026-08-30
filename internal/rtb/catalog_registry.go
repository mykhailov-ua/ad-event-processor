package rtb

import (
	"sync/atomic"
)

// Registry owns the hot-path catalog snapshot and BudgetStore slot map.
// Cold path (UpdateCampaigns) builds immutable SoA registries per geo shard and publishes via atomic.Pointer;
// RunAuction / RunAuctionEval load once per request with no mutex.
type Registry struct {
	// catalog: [64]*CampaignAuctionRegistry keyed by GeoHashVal & geoShardMask (63).
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
	for i := range geoShardCount {
		empty.shards[i] = &CampaignAuctionRegistry{}
	}
	registry.catalog.Store(empty)
	return registry
}

func (r *Registry) SetClearingMode(mode ClearingMode) {
	r.clearingMode.Store(uint32(mode))
}

func (r *Registry) ClearingMode() ClearingMode {
	return ClearingMode(r.clearingMode.Load())
}

func (r *Registry) SetTargetingIndexEnabled(enabled bool) {
	r.targetingIndexEnabled.Store(enabled)
}

func (r *Registry) TargetingIndexEnabled() bool {
	return r.targetingIndexEnabled.Load()
}

// LoadShard returns the SoA registry for req.GeoHash & geoShardMask (64-way geo partition).
func (r *Registry) LoadShard(idx uint32) *CampaignAuctionRegistry {
	snap := r.catalog.Load()
	if snap == nil {
		return nil
	}
	return snap.shards[idx&geoShardMask]
}

func (r *Registry) loadCatalog() *catalogSnapshot {
	return r.catalog.Load()
}

func (r *Registry) Store() *BudgetStore {
	return r.store
}

func (r *Registry) SetFcapSnapshot(snap *FcapSnapshot) {
	r.fcap.store(snap)
}

func (r *Registry) LoadFcapSnapshot() *FcapSnapshot {
	return r.fcap.load()
}

func (r *Registry) UpdateCreatives(creatives []CreativeData) {
	if len(creatives) == 0 {
		r.pendingCreatives = nil
		return
	}
	r.pendingCreatives = append(r.pendingCreatives[:0], creatives...)
}

// UpdateCampaigns cold-rebuilds columnar CampaignAuctionRegistry slices per geo shard,
// materializes GeoBucketSoA/TargetBucketSoA, presorts buckets, then atomically swaps catalog.
// Hot readers never observe partial rebuilds.
func (r *Registry) UpdateCampaigns(campaigns []CampaignData) {
	var counts [geoShardCount]int
	for i := range campaigns {
		shardIdx := campaigns[i].GeoHashVal & geoShardMask
		counts[shardIdx]++
	}

	var registries [geoShardCount]*CampaignAuctionRegistry
	for shardIdx := range geoShardCount {
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
		// BudgetIndices map into BudgetStore CAS slices; consumed by CheckAndSpendAll when
		// RTB_BUDGET_AUTHORITY=rtb and RunAuction(spend=true). Shadow RunAuctionEval skips spend.
		reg.BudgetIndices[wIdx] = r.store.GetOrAllocateSlot(c.ID, c.Budget)
		reg.CustomerBudgetIndices[wIdx] = r.store.GetOrAllocateCustomerSlot(c.CustomerID, c.CustomerBudget)
		reg.DaypartMasks[wIdx] = c.DaypartMask
		reg.TZOffsetSec[wIdx] = c.TZOffsetSec
		reg.ScheduleStart[wIdx] = c.ScheduleStart
		reg.ScheduleEnd[wIdx] = c.ScheduleEnd
		reg.FreqLimits[wIdx] = c.FreqLimit
		reg.FcapPrefixHash[wIdx] = c.FcapPrefixHash

		writeIndices[shardIdx]++
	}

	targetingEnabled := r.targetingIndexEnabled.Load()
	shardCreatives := partitionCreativesByShard(campaigns, r.pendingCreatives)
	for shardIdx := range geoShardCount {
		buildCreativeCache(registries[shardIdx], shardCreatives[shardIdx])
		// Geo buckets: campaigns grouped by GeoHashVal; GeoBucketSoA holds denormalized rows for linear scan.
		buildGeoIndex(registries[shardIdx])
		if targetingEnabled {
			buildTargetingIndex(registries[shardIdx])
		}
		// Presort by effective score so rankCandidates can early-break below floor.
		sortRegistryBuckets(registries[shardIdx])
	}

	r.publishCatalog(registries)
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

// publishCatalog swaps the immutable catalogSnapshot; never mutate a published registry in place.
func (r *Registry) publishCatalog(shards [geoShardCount]*CampaignAuctionRegistry) {
	r.catalog.Store(&catalogSnapshot{shards: shards})
	r.snapGen.Add(1)
}
