package rtb

func (r *Registry) RunAuction(req *BidRequest) (AuctionResult, NoBidReason) {
	return r.runAuction(req, true)
}

func (r *Registry) RunAuctionEval(req *BidRequest) (AuctionResult, NoBidReason) {
	return r.runAuction(req, false)
}

func (r *Registry) runAuction(req *BidRequest, spend bool) (AuctionResult, NoBidReason) {
	start := auctionStartMono()

	if req == nil || req.MinBid < 0 {
		recordAuctionOutcome(start, NoBidInvalidRequest, 0)
		return AuctionResult{}, NoBidInvalidRequest
	}
	reg := r.LoadShard(req.GeoHash)
	if reg == nil || reg.Count == 0 {
		recordAuctionOutcome(start, NoBidEmptyShard, 0)
		return AuctionResult{}, NoBidEmptyShard
	}

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

	if spend {
		winnerBudgetIdx := reg.BudgetIndices[winnerIdx]
		customerIdx := reg.CustomerBudgetIndices[winnerIdx]
		dailyLimit := reg.DailyBudgets[winnerIdx]
		if !r.store.CheckAndSpendAll(winnerBudgetIdx, customerIdx, price, dailyLimit) {
			recordAuctionOutcome(start, NoBidSpendFailed, scanned)
			return AuctionResult{}, NoBidSpendFailed
		}
		recordBudgetSpendMirror(reg.CampaignIDs[winnerIdx], winnerBudgetIdx, price)
	}

	recordAuctionOutcome(start, NoBidNone, scanned)
	return AuctionResult{
		CampaignID: reg.CampaignIDs[winnerIdx],
		CreativeID: winnerCreative,
		Price:      price,
	}, NoBidNone
}
