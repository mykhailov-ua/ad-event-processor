package rtb

const (
	rankDeadlineCheckMask = 127
	rankMaxScanCandidates = 500
)

func GeoBitFromHash(geoHash uint32) uint64 {
	return uint64(1) << (geoHash & 63)
}

func (registry *Registry) catalogSlicesValid(reg *CampaignAuctionRegistry) bool {
	count := reg.Count
	if !(count <= len(reg.CampaignIDs) && count <= len(reg.Bids) &&
		count <= len(reg.CTRPPM) && count <= len(reg.Reserves) &&
		count <= len(reg.DailyBudgets) && count <= len(reg.PacingOpen) &&
		count <= len(reg.DeviceMasks) && count <= len(reg.CategoryMasks) &&
		count <= len(reg.GeoHashes) && count <= len(reg.Weights) &&
		count <= len(reg.BoostPPM) &&
		count <= len(reg.BudgetIndices) && count <= len(reg.CustomerBudgetIndices)) {
		return false
	}
	geoEnd := reg.GeoBucketSoA.len()
	if geoEnd > 0 && !reg.GeoBucketSoA.slicesValid(geoEnd) {
		return false
	}
	targetEnd := reg.TargetBucketSoA.len()
	if targetEnd > 0 && !reg.TargetBucketSoA.slicesValid(targetEnd) {
		return false
	}
	return true
}

func bidsAt(reg *CampaignAuctionRegistry, idx int) int64 {
	return reg.Bids[idx]
}

func (registry *Registry) candidateRange(
	reg *CampaignAuctionRegistry,
	req *BidRequest,
) (soa *candidateBucketSoA, start int, end int, ok bool) {
	if registry.targetingIndexEnabled.Load() {
		if start, end, ok = reg.targetingRange(req.GeoHash, req.DeviceType, req.CategoryMask); ok {
			return &reg.TargetBucketSoA, start, end, true
		}
	}
	start, end, ok = reg.geoRange(req.GeoHash)
	return &reg.GeoBucketSoA, start, end, ok
}

func (registry *Registry) rankCandidates(
	reg *CampaignAuctionRegistry,
	req *BidRequest,
	soa *candidateBucketSoA,
	bucketStart int,
	bucketEnd int,
) (winnerIdx int, winnerCreative CreativeID, secondBid int64, scanned int, noBid NoBidReason) {
	winnerIdx = -1
	winnerCreative = 0
	var maxScore int64 = -1
	secondBid = -1
	var pacingBlocked bool
	var dailyBlocked bool
	var daypartBlocked bool
	var freqBlocked bool

	if soa == nil || !soa.slicesValid(bucketEnd) {
		return -1, 0, -1, 0, NoBidCorruptCatalog
	}
	if bucketStart < 0 || bucketEnd < bucketStart || bucketEnd > soa.len() {
		return -1, 0, -1, 0, NoBidCorruptCatalog
	}

	catalogIdx := soa.CatalogIdx[bucketStart:bucketEnd]
	creativeIDs := soa.CreativeIDs[bucketStart:bucketEnd]
	bids := soa.Bids[bucketStart:bucketEnd]
	ctrppm := soa.CTRPPM[bucketStart:bucketEnd]
	reserves := soa.Reserves[bucketStart:bucketEnd]
	dailyBudgets := soa.DailyBudgets[bucketStart:bucketEnd]
	pacingOpen := soa.PacingOpen[bucketStart:bucketEnd]
	deviceMasks := soa.DeviceMasks[bucketStart:bucketEnd]
	categoryMasks := soa.CategoryMasks[bucketStart:bucketEnd]
	weights := soa.Weights[bucketStart:bucketEnd]
	boostPPM := soa.BoostPPM[bucketStart:bucketEnd]
	mediaTypes := soa.MediaTypes[bucketStart:bucketEnd]
	durationSec := soa.DurationSec[bucketStart:bucketEnd]
	budgetIndices := soa.BudgetIndices[bucketStart:bucketEnd]
	customerBudgetIndices := soa.CustomerBudgetIndices[bucketStart:bucketEnd]
	daypartMasks := soa.DaypartMasks[bucketStart:bucketEnd]
	tzOffsets := soa.TZOffsetSec[bucketStart:bucketEnd]
	scheduleStarts := soa.ScheduleStart[bucketStart:bucketEnd]
	scheduleEnds := soa.ScheduleEnd[bucketStart:bucketEnd]
	freqLimits := soa.FreqLimits[bucketStart:bucketEnd]
	fcapPrefixHashes := soa.FcapPrefixHash[bucketStart:bucketEnd]

	regCount := reg.Count
	deviceType := req.DeviceType
	categoryMask := req.CategoryMask
	mediaMask := req.MediaTypeMask
	maxDuration := req.MaxDurationSec
	minBid := req.MinBid
	store := registry.store
	winnerBid := int64(-1)
	winnerWeight := uint32(0)
	deadline := req.DeadlineMono
	hasDeadline := deadline > 0
	nowUnix := req.NowUnix
	fcapUserHash := req.FcapUserHash
	fcapSnap := registry.LoadFcapSnapshot()

	n := len(catalogIdx)
	if n > 0 {
		_ = catalogIdx[n-1]
		_ = creativeIDs[n-1]
		_ = bids[n-1]
		_ = weights[n-1]
	}

	for pos := range n {
		if hasDeadline && scanned&rankDeadlineCheckMask == 0 && monotonicNano() > deadline {
			return -1, 0, -1, scanned, NoBidTimeout
		}

		start := scheduleStarts[pos]
		if start > 0 && nowUnix < start {
			daypartBlocked = true
			continue
		}
		end := scheduleEnds[pos]
		if end > 0 && nowUnix >= end {
			daypartBlocked = true
			continue
		}
		mask := daypartMasks[pos]
		if mask != 0 {
			tzOff := tzOffsets[pos]
			local := nowUnix + int64(tzOff)
			sec := local % 86400
			if sec < 0 {
				sec += 86400
			}
			hour := int(sec / 3600)
			if hour >= 0 && hour < 24 && (mask&(1<<uint(hour)) == 0) {
				daypartBlocked = true
				continue
			}
		}

		freqLimit := freqLimits[pos]
		if freqLimit > 0 && fcapUserHash != 0 && fcapSnap != nil {
			prefixHash := fcapPrefixHashes[pos]
			if prefixHash != 0 {
				if cnt, ok := fcapSnap.FcapCount(prefixHash, fcapUserHash); ok && FreqCapExceeded(freqLimit, cnt) {
					freqBlocked = true
					continue
				}
			}
		}

		scanned++
		if scanned > rankMaxScanCandidates {
			return -1, 0, -1, scanned, NoBidScanLimit
		}

		i := int(catalogIdx[pos])
		if i < 0 || i >= regCount {
			return -1, 0, -1, scanned, NoBidCorruptCatalog
		}

		if pacingOpen[pos] == PacingClosed {
			pacingBlocked = true
			continue
		}
		if (deviceMasks[pos] & deviceType) == 0 {
			continue
		}
		if (categoryMasks[pos] & categoryMask) == 0 {
			continue
		}
		if blocked := req.BlockedCatMask; blocked != 0 && (categoryMasks[pos]&blocked) != 0 {
			continue
		}
		if mediaMask != 0 && mediaTypes[pos]&mediaMask == 0 {
			continue
		}
		if maxDuration > 0 {
			dur := durationSec[pos]
			if dur > 0 && dur > maxDuration {
				continue
			}
		}

		bid := bids[pos]
		reserve := reserves[pos]
		if bid < reserve || bid < minBid {
			continue
		}

		budgetIdx := budgetIndices[pos]
		if !store.budgetSlotExists(budgetIdx) {
			return -1, 0, -1, scanned, NoBidCorruptCatalog
		}
		if store.LoadBudget(budgetIdx) < bid {
			continue
		}
		if dailyBudgets[pos] > 0 && store.loadDailyHeadroom(budgetIdx, dailyBudgets[pos]) < bid {
			dailyBlocked = true
			continue
		}
		customerIdx := customerBudgetIndices[pos]
		if customerIdx != invalidCustomerBudgetIdx && store.LoadCustomerBudget(customerIdx) < bid {
			continue
		}

		score := effectiveScoreWithBoost(bid, ctrppm[pos], boostPPM[pos])
		if winnerIdx >= 0 && secondBid >= 0 && score < maxScore {
			break
		}
		if score > maxScore {
			if winnerIdx >= 0 {
				secondBid = winnerBid
			}
			maxScore = score
			winnerIdx = i
			winnerBid = bid
			winnerWeight = weights[pos]
			winnerCreative = creativeIDs[pos]
		} else if score == maxScore && winnerIdx >= 0 {
			if weights[pos] > winnerWeight {
				secondBid = winnerBid
				winnerIdx = i
				winnerBid = bid
				winnerWeight = weights[pos]
				winnerCreative = creativeIDs[pos]
			}
			if bid > secondBid {
				secondBid = bid
			}
		} else if winnerIdx >= 0 && bid > secondBid {
			secondBid = bid
		}
	}

	if winnerIdx == -1 {
		if daypartBlocked {
			return -1, 0, -1, scanned, NoBidDaypartClosed
		}
		if freqBlocked {
			return -1, 0, -1, scanned, NoBidFreqCapExceeded
		}
		if pacingBlocked {
			return -1, 0, -1, scanned, NoBidPacingClosed
		}
		if dailyBlocked {
			return -1, 0, -1, scanned, NoBidDailyCapExceeded
		}
		return -1, 0, -1, scanned, NoBidNoCandidates
	}
	return winnerIdx, winnerCreative, secondBid, scanned, NoBidNone
}
