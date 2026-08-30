package rtb

// RunAuction ranks the SoA catalog, clears price, and debits BudgetStore via
// CheckAndSpendAll (CAS). Used when RTB_MODE=live and RTB_BUDGET_AUTHORITY=rtb.
func (r *Registry) RunAuction(req *BidRequest) (AuctionResult, NoBidReason) {
	return r.runAuction(req, true)
}

// RunAuctionEval runs the same rank and clearing path without spend. Shadow mode
// and admin validate-bid-request use this to diff winners without mutating budgets.
func (r *Registry) RunAuctionEval(req *BidRequest) (AuctionResult, NoBidReason) {
	return r.runAuction(req, false)
}

func (r *Registry) runAuction(req *BidRequest, spend bool) (AuctionResult, NoBidReason) {
	start := auctionStartMono()

	if req == nil || req.MinBid < 0 {
		recordAuctionOutcome(start, NoBidInvalidRequest, 0)
		return AuctionResult{}, NoBidInvalidRequest
	}
	// LoadShard: atomic.Pointer[catalogSnapshot] read; geo hash selects one of 64
	// immutable CampaignAuctionRegistry shards rebuilt on cold-path catalog reload.
	reg := r.LoadShard(req.GeoHash)
	if reg == nil || reg.Count == 0 {
		recordAuctionOutcome(start, NoBidEmptyShard, 0)
		return AuctionResult{}, NoBidEmptyShard
	}

	// Guard parallel SoA slice lengths before rankCandidates subviews the bucket.
	if !r.catalogSlicesValid(reg) {
		recordAuctionOutcome(start, NoBidCorruptCatalog, reg.Count)
		return AuctionResult{}, NoBidCorruptCatalog
	}

	if req.DealBlock != NoBidNone {
		recordAuctionOutcome(start, req.DealBlock, 0)
		return AuctionResult{}, req.DealBlock
	}

	bucket, bucketStart, bucketEnd, ok := r.candidateRange(reg, req)
	if !ok {
		recordAuctionOutcome(start, NoBidNoCandidates, 0)
		return AuctionResult{}, NoBidNoCandidates
	}

	clearing := r.ClearingMode()
	// rankCandidates: linear scan over presorted candidateBucketSoA; scanned capped
	// at rankMaxScanCandidates (500, core.mdc RTB p99). Zero heap on hot path.
	winnerIdx, winnerCreative, secondBid, scanned, noBid := r.rankCandidates(reg, req, bucket, bucketStart, bucketEnd)
	if noBid != NoBidNone {
		recordAuctionOutcome(start, noBid, scanned)
		return AuctionResult{}, noBid
	}

	price := r.clearingPrice(clearing, req.MinBid, bidsAt(reg, winnerIdx), secondBid)
	price = applyReserve(price, reg.Reserves[winnerIdx], bidsAt(reg, winnerIdx))

	if winnerIdx >= len(reg.BudgetIndices) || winnerIdx >= len(reg.CampaignIDs) {
		recordAuctionOutcome(start, NoBidCorruptCatalog, scanned)
		return AuctionResult{}, NoBidCorruptCatalog
	}

	// spend=false skips CAS debit; rank still used LoadBudget (atomic load) read-only.
	if spend {
		winnerBudgetIdx := reg.BudgetIndices[winnerIdx]
		customerIdx := reg.CustomerBudgetIndices[winnerIdx]
		dailyLimit := reg.DailyBudgets[winnerIdx]
		// CheckAndSpendAll: campaign + optional customer + daily caps via CAS loops
		// in budget_spend.go; rolls back prior legs on failure (NoBidSpendFailed).
		if !r.store.CheckAndSpendAll(winnerBudgetIdx, customerIdx, price, dailyLimit) {
			recordAuctionOutcome(start, NoBidSpendFailed, scanned)
			return AuctionResult{}, NoBidSpendFailed
		}
		// Async Redis mirror when authority=rtb; does not participate in auction CAS.
		recordBudgetSpendMirror(reg.CampaignIDs[winnerIdx], winnerBudgetIdx, price)
	}

	recordAuctionOutcome(start, NoBidNone, scanned)
	return AuctionResult{
		CampaignID: reg.CampaignIDs[winnerIdx],
		CreativeID: winnerCreative,
		Price:      price,
	}, NoBidNone
}
