package rtb

// candidateBucketSoA stores hot-path candidate fields in bucket iteration order.
// All slices share the same length; populated on the cold catalog rebuild path.
type candidateBucketSoA struct {
	CatalogIdx            []uint32
	CreativeIDs           []CreativeID
	Bids                  []int64
	CTRPPM                []uint32
	Reserves              []int64
	DailyBudgets          []int64
	PacingOpen            []uint8
	DeviceMasks           []uint8
	CategoryMasks         []uint64
	Weights               []uint32
	BoostPPM              []uint32
	MediaTypes            []uint8
	DurationSec           []uint32
	BudgetIndices         []uint32
	CustomerBudgetIndices []uint32
	DaypartMasks          []uint32
	TZOffsetSec           []int32
	ScheduleStart         []int64
	ScheduleEnd           []int64
	FreqLimits            []uint32
	FcapPrefixHash        []uint64
}

func (soa *candidateBucketSoA) len() int {
	if soa == nil {
		return 0
	}
	return len(soa.CatalogIdx)
}

func (soa *candidateBucketSoA) slicesValid(end int) bool {
	if soa == nil || end < 0 || end > len(soa.CatalogIdx) {
		return false
	}
	return end <= len(soa.CreativeIDs) &&
		end <= len(soa.Bids) &&
		end <= len(soa.CTRPPM) &&
		end <= len(soa.Reserves) &&
		end <= len(soa.DailyBudgets) &&
		end <= len(soa.PacingOpen) &&
		end <= len(soa.DeviceMasks) &&
		end <= len(soa.CategoryMasks) &&
		end <= len(soa.Weights) &&
		end <= len(soa.BoostPPM) &&
		end <= len(soa.MediaTypes) &&
		end <= len(soa.DurationSec) &&
		end <= len(soa.BudgetIndices) &&
		end <= len(soa.CustomerBudgetIndices)
}

func resetBucketSoA(soa *candidateBucketSoA) {
	if soa == nil {
		return
	}
	soa.CatalogIdx = soa.CatalogIdx[:0]
	soa.CreativeIDs = soa.CreativeIDs[:0]
	soa.Bids = soa.Bids[:0]
	soa.CTRPPM = soa.CTRPPM[:0]
	soa.Reserves = soa.Reserves[:0]
	soa.DailyBudgets = soa.DailyBudgets[:0]
	soa.PacingOpen = soa.PacingOpen[:0]
	soa.DeviceMasks = soa.DeviceMasks[:0]
	soa.CategoryMasks = soa.CategoryMasks[:0]
	soa.Weights = soa.Weights[:0]
	soa.BoostPPM = soa.BoostPPM[:0]
	soa.MediaTypes = soa.MediaTypes[:0]
	soa.DurationSec = soa.DurationSec[:0]
	soa.BudgetIndices = soa.BudgetIndices[:0]
	soa.CustomerBudgetIndices = soa.CustomerBudgetIndices[:0]
	soa.DaypartMasks = soa.DaypartMasks[:0]
	soa.TZOffsetSec = soa.TZOffsetSec[:0]
	soa.ScheduleStart = soa.ScheduleStart[:0]
	soa.ScheduleEnd = soa.ScheduleEnd[:0]
	soa.FreqLimits = soa.FreqLimits[:0]
	soa.FcapPrefixHash = soa.FcapPrefixHash[:0]
}

func appendBucketCandidate(soa *candidateBucketSoA, reg *CampaignAuctionRegistry, catalogIdx uint32) {
	crStart, crEnd, hasCreatives := campaignCreativeRange(reg, int(catalogIdx))
	if hasCreatives {
		cr := &reg.CreativeCache
		for slot := crStart; slot < crEnd; slot++ {
			appendBucketRow(soa, reg, catalogIdx, cr.CreativeIDs[slot], cr.Bids[slot], cr.CTRPPM[slot], cr.Weights[slot], cr.MediaTypes[slot], cr.DurationSec[slot])
		}
		return
	}
	appendBucketRow(soa, reg, catalogIdx, 0, reg.Bids[int(catalogIdx)], reg.CTRPPM[int(catalogIdx)], reg.Weights[int(catalogIdx)], 0, 0)
}

func appendBucketRow(
	soa *candidateBucketSoA,
	reg *CampaignAuctionRegistry,
	catalogIdx uint32,
	creativeID CreativeID,
	bid int64,
	ctr uint32,
	weight uint32,
	media uint8,
	duration uint32,
) {
	i := int(catalogIdx)
	soa.CatalogIdx = append(soa.CatalogIdx, catalogIdx)
	soa.CreativeIDs = append(soa.CreativeIDs, creativeID)
	soa.Bids = append(soa.Bids, bid)
	soa.CTRPPM = append(soa.CTRPPM, ctr)
	soa.Reserves = append(soa.Reserves, reg.Reserves[i])
	soa.DailyBudgets = append(soa.DailyBudgets, reg.DailyBudgets[i])
	soa.PacingOpen = append(soa.PacingOpen, reg.PacingOpen[i])
	soa.DeviceMasks = append(soa.DeviceMasks, reg.DeviceMasks[i])
	soa.CategoryMasks = append(soa.CategoryMasks, reg.CategoryMasks[i])
	soa.Weights = append(soa.Weights, weight)
	soa.BoostPPM = append(soa.BoostPPM, reg.BoostPPM[i])
	soa.MediaTypes = append(soa.MediaTypes, media)
	soa.DurationSec = append(soa.DurationSec, duration)
	soa.BudgetIndices = append(soa.BudgetIndices, reg.BudgetIndices[i])
	soa.CustomerBudgetIndices = append(soa.CustomerBudgetIndices, reg.CustomerBudgetIndices[i])
	soa.DaypartMasks = append(soa.DaypartMasks, sliceAtU32(reg.DaypartMasks, i))
	soa.TZOffsetSec = append(soa.TZOffsetSec, sliceAtI32(reg.TZOffsetSec, i))
	soa.ScheduleStart = append(soa.ScheduleStart, sliceAtI64(reg.ScheduleStart, i))
	soa.ScheduleEnd = append(soa.ScheduleEnd, sliceAtI64(reg.ScheduleEnd, i))
	soa.FreqLimits = append(soa.FreqLimits, sliceAtU32(reg.FreqLimits, i))
	soa.FcapPrefixHash = append(soa.FcapPrefixHash, sliceAtU64(reg.FcapPrefixHash, i))
}

func sliceAtU32(s []uint32, i int) uint32 {
	if i < 0 || i >= len(s) {
		return 0
	}
	return s[i]
}

func sliceAtI32(s []int32, i int) int32 {
	if i < 0 || i >= len(s) {
		return 0
	}
	return s[i]
}

func sliceAtI64(s []int64, i int) int64 {
	if i < 0 || i >= len(s) {
		return 0
	}
	return s[i]
}

func sliceAtU64(s []uint64, i int) uint64 {
	if i < 0 || i >= len(s) {
		return 0
	}
	return s[i]
}

func ensureBucketSoACap(soa *candidateBucketSoA, wantCap int) {
	if wantCap <= 0 {
		return
	}
	if cap(soa.CatalogIdx) < wantCap {
		soa.CatalogIdx = make([]uint32, 0, wantCap)
		soa.CreativeIDs = make([]CreativeID, 0, wantCap)
		soa.Bids = make([]int64, 0, wantCap)
		soa.CTRPPM = make([]uint32, 0, wantCap)
		soa.Reserves = make([]int64, 0, wantCap)
		soa.DailyBudgets = make([]int64, 0, wantCap)
		soa.PacingOpen = make([]uint8, 0, wantCap)
		soa.DeviceMasks = make([]uint8, 0, wantCap)
		soa.CategoryMasks = make([]uint64, 0, wantCap)
		soa.Weights = make([]uint32, 0, wantCap)
		soa.BoostPPM = make([]uint32, 0, wantCap)
		soa.MediaTypes = make([]uint8, 0, wantCap)
		soa.DurationSec = make([]uint32, 0, wantCap)
		soa.BudgetIndices = make([]uint32, 0, wantCap)
		soa.CustomerBudgetIndices = make([]uint32, 0, wantCap)
		soa.DaypartMasks = make([]uint32, 0, wantCap)
		soa.TZOffsetSec = make([]int32, 0, wantCap)
		soa.ScheduleStart = make([]int64, 0, wantCap)
		soa.ScheduleEnd = make([]int64, 0, wantCap)
		soa.FreqLimits = make([]uint32, 0, wantCap)
		soa.FcapPrefixHash = make([]uint64, 0, wantCap)
	}
}

func swapBucketSoA(soa *candidateBucketSoA, i, j int) {
	soa.CatalogIdx[i], soa.CatalogIdx[j] = soa.CatalogIdx[j], soa.CatalogIdx[i]
	soa.CreativeIDs[i], soa.CreativeIDs[j] = soa.CreativeIDs[j], soa.CreativeIDs[i]
	soa.Bids[i], soa.Bids[j] = soa.Bids[j], soa.Bids[i]
	soa.CTRPPM[i], soa.CTRPPM[j] = soa.CTRPPM[j], soa.CTRPPM[i]
	soa.Reserves[i], soa.Reserves[j] = soa.Reserves[j], soa.Reserves[i]
	soa.DailyBudgets[i], soa.DailyBudgets[j] = soa.DailyBudgets[j], soa.DailyBudgets[i]
	soa.PacingOpen[i], soa.PacingOpen[j] = soa.PacingOpen[j], soa.PacingOpen[i]
	soa.DeviceMasks[i], soa.DeviceMasks[j] = soa.DeviceMasks[j], soa.DeviceMasks[i]
	soa.CategoryMasks[i], soa.CategoryMasks[j] = soa.CategoryMasks[j], soa.CategoryMasks[i]
	soa.Weights[i], soa.Weights[j] = soa.Weights[j], soa.Weights[i]
	soa.BoostPPM[i], soa.BoostPPM[j] = soa.BoostPPM[j], soa.BoostPPM[i]
	soa.MediaTypes[i], soa.MediaTypes[j] = soa.MediaTypes[j], soa.MediaTypes[i]
	soa.DurationSec[i], soa.DurationSec[j] = soa.DurationSec[j], soa.DurationSec[i]
	soa.BudgetIndices[i], soa.BudgetIndices[j] = soa.BudgetIndices[j], soa.BudgetIndices[i]
	soa.CustomerBudgetIndices[i], soa.CustomerBudgetIndices[j] = soa.CustomerBudgetIndices[j], soa.CustomerBudgetIndices[i]
	soa.DaypartMasks[i], soa.DaypartMasks[j] = soa.DaypartMasks[j], soa.DaypartMasks[i]
	soa.TZOffsetSec[i], soa.TZOffsetSec[j] = soa.TZOffsetSec[j], soa.TZOffsetSec[i]
	soa.ScheduleStart[i], soa.ScheduleStart[j] = soa.ScheduleStart[j], soa.ScheduleStart[i]
	soa.ScheduleEnd[i], soa.ScheduleEnd[j] = soa.ScheduleEnd[j], soa.ScheduleEnd[i]
	soa.FreqLimits[i], soa.FreqLimits[j] = soa.FreqLimits[j], soa.FreqLimits[i]
	soa.FcapPrefixHash[i], soa.FcapPrefixHash[j] = soa.FcapPrefixHash[j], soa.FcapPrefixHash[i]
}
