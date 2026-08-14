package rtb

import (
	"github.com/bidshard/ad-event-processor/pkg/faultproof"

	"math"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func faultInstallShard(reg *Registry, geo uint32, shard *CampaignAuctionRegistry) {
	snap := reg.loadCatalog()
	var shards [geoShardCount]*CampaignAuctionRegistry
	if snap != nil {
		shards = snap.shards
	}
	shards[geo&geoShardMask] = shard
	reg.publishCatalog(shards)
}

func faultRunAuction(reg *Registry, req *BidRequest) (res AuctionResult, reason NoBidReason, panicked bool) {
	defer func() {
		if recover() != nil {
			panicked = true
		}
	}()
	res, reason = reg.RunAuction(req)
	return res, reason, false
}

func faultBudgetTriple(store *BudgetStore, campIdx, custIdx uint32) (campaign, customer, daily int64) {
	campaign = store.LoadBudget(campIdx)
	customer = store.LoadCustomerBudget(custIdx)
	daily = store.loadOn(&store.dailySpent, campIdx)
	return campaign, customer, daily
}

func TestFault_NilRequest(t *testing.T) {
	store := NewBudgetStore()
	reg := NewRegistry(store)
	reg.UpdateCampaigns(singleCampaign(CampaignID(1), 100, 1000))

	_, reason, panicked := faultRunAuction(reg, nil)
	assert.False(t, panicked)
	assert.Equal(t, NoBidInvalidRequest, reason)
	assert.Equal(t, int64(1000), store.GetBudget(CampaignID(1)))
	faultproof.Log(t, "rtb_nil_request", map[string]string{"outcome": "invalid", "no_panic": "true"})
}

func TestFault_NegativeMinBid(t *testing.T) {
	store := NewBudgetStore()
	reg := NewRegistry(store)
	reg.UpdateCampaigns(singleCampaign(CampaignID(1), 100, 1000))

	_, reason, panicked := faultRunAuction(reg, &BidRequest{
		DeviceType: 1, CategoryMask: 1, GeoHash: 7, MinBid: -1,
	})
	assert.False(t, panicked)
	assert.Equal(t, NoBidInvalidRequest, reason)
	faultproof.Log(t, "rtb_negative_min_bid", map[string]string{"outcome": "invalid"})
}

func TestFault_ZeroDeviceMask(t *testing.T) {
	store := NewBudgetStore()
	reg := NewRegistry(store)
	reg.UpdateCampaigns(singleCampaign(CampaignID(1), 100, 1000))

	_, reason, panicked := faultRunAuction(reg, &BidRequest{
		DeviceType: 0, CategoryMask: 1, GeoHash: 7, MinBid: 50,
	})
	assert.False(t, panicked)
	assert.Equal(t, NoBidNoCandidates, reason)
	faultproof.Log(t, "rtb_zero_device", map[string]string{"outcome": "no_candidates"})
}

func TestFault_ZeroCategoryMask(t *testing.T) {
	store := NewBudgetStore()
	reg := NewRegistry(store)
	reg.UpdateCampaigns(singleCampaign(CampaignID(1), 100, 1000))

	_, reason, panicked := faultRunAuction(reg, &BidRequest{
		DeviceType: 1, CategoryMask: 0, GeoHash: 7, MinBid: 50,
	})
	assert.False(t, panicked)
	assert.Equal(t, NoBidNoCandidates, reason)
	faultproof.Log(t, "rtb_zero_category", map[string]string{"outcome": "no_candidates"})
}

func TestFault_MaxIntMinBid(t *testing.T) {
	store := NewBudgetStore()
	reg := NewRegistry(store)
	reg.UpdateCampaigns(singleCampaign(CampaignID(1), 100, 1000))

	_, reason, panicked := faultRunAuction(reg, &BidRequest{
		DeviceType: 1, CategoryMask: 1, GeoHash: 7, MinBid: math.MaxInt64,
	})
	assert.False(t, panicked)
	assert.False(t, reason.OK())
	faultproof.Log(t, "rtb_max_min_bid", map[string]string{"outcome": reason.String()})
}

func TestFault_EmptyGeoShard(t *testing.T) {
	store := NewBudgetStore()
	reg := NewRegistry(store)
	reg.UpdateCampaigns(singleCampaign(CampaignID(1), 100, 1000))

	_, reason, panicked := faultRunAuction(reg, stdReq(9999, 50))
	assert.False(t, panicked)
	assert.True(t, reason == NoBidEmptyShard || reason == NoBidNoCandidates)
	faultproof.Log(t, "rtb_unknown_geo", map[string]string{"outcome": reason.String()})
}

func TestFault_CountExceedsSlices(t *testing.T) {
	store := NewBudgetStore()
	reg := NewRegistry(store)
	reg.UpdateCampaigns(singleCampaign(CampaignID(1), 100, 1000))

	faultInstallShard(reg, 7, &CampaignAuctionRegistry{
		Count:       5,
		CampaignIDs: []CampaignID{1},
		Bids:        []int64{100},
	})

	_, reason, panicked := faultRunAuction(reg, stdReq(7, 50))
	assert.False(t, panicked, "corrupt count must not panic")
	assert.Equal(t, NoBidCorruptCatalog, reason)
	faultproof.Log(t, "rtb_count_gt_slices", map[string]string{"outcome": "corrupt_catalog"})
}

func TestFault_NonemptyCountEmptyGeoIndex(t *testing.T) {
	store := NewBudgetStore()
	reg := NewRegistry(store)
	idx := store.GetOrAllocateSlot(CampaignID(1), 1000)

	faultInstallShard(reg, 7, &CampaignAuctionRegistry{
		Count:                 1,
		CampaignIDs:           []CampaignID{1},
		Bids:                  []int64{100},
		CTRPPM:                []uint32{CTRPPMUnit},
		Reserves:              []int64{0},
		DailyBudgets:          []int64{0},
		PacingOpen:            []uint8{PacingOpen},
		DeviceMasks:           []uint8{1},
		CategoryMasks:         []uint64{1},
		GeoHashes:             []uint32{7},
		Weights:               []uint32{1},
		BoostPPM:              []uint32{CTRPPMUnit},
		BudgetIndices:         []uint32{idx},
		CustomerBudgetIndices: []uint32{invalidCustomerBudgetIdx},
		GeoBucketCount:        0,
	})

	_, reason, panicked := faultRunAuction(reg, stdReq(7, 50))
	assert.False(t, panicked)
	assert.Equal(t, NoBidNoCandidates, reason)
	faultproof.Log(t, "rtb_empty_geo_index", map[string]string{"outcome": "no_candidates"})
}

func TestFault_GeoBucketIdxOutOfBounds(t *testing.T) {
	store := NewBudgetStore()
	reg := NewRegistry(store)
	idx := store.GetOrAllocateSlot(CampaignID(1), 1000)

	faultInstallShard(reg, 7, &CampaignAuctionRegistry{
		Count:                 1,
		CampaignIDs:           []CampaignID{1},
		Bids:                  []int64{100},
		CTRPPM:                []uint32{CTRPPMUnit},
		Reserves:              []int64{0},
		DailyBudgets:          []int64{0},
		PacingOpen:            []uint8{PacingOpen},
		DeviceMasks:           []uint8{1},
		CategoryMasks:         []uint64{1},
		GeoHashes:             []uint32{7},
		Weights:               []uint32{1},
		BoostPPM:              []uint32{CTRPPMUnit},
		BudgetIndices:         []uint32{idx},
		CustomerBudgetIndices: []uint32{invalidCustomerBudgetIdx},
		GeoBucketCount:        1,
		GeoBucketHash:         []uint32{7},
		GeoBucketStart:        []uint32{0, 1},
		GeoBucketSoA: candidateBucketSoA{
			CatalogIdx:            []uint32{99},
			Bids:                  []int64{100},
			CTRPPM:                []uint32{CTRPPMUnit},
			Reserves:              []int64{0},
			DailyBudgets:          []int64{0},
			PacingOpen:            []uint8{PacingOpen},
			DeviceMasks:           []uint8{1},
			CategoryMasks:         []uint64{1},
			Weights:               []uint32{1},
			BoostPPM:              []uint32{CTRPPMUnit},
			BudgetIndices:         []uint32{idx},
			CustomerBudgetIndices: []uint32{invalidCustomerBudgetIdx},
		},
	})

	_, reason, panicked := faultRunAuction(reg, stdReq(7, 50))
	assert.False(t, panicked, "OOB geo bucket index must not panic")
	assert.Equal(t, NoBidCorruptCatalog, reason)
	faultproof.Log(t, "rtb_oob_geo_bucket_idx", map[string]string{"outcome": "corrupt_catalog"})
}

func TestFault_BudgetIndexOutOfRange(t *testing.T) {
	store := NewBudgetStore()
	reg := NewRegistry(store)

	faultInstallShard(reg, 7, &CampaignAuctionRegistry{
		Count:                 1,
		CampaignIDs:           []CampaignID{1},
		Bids:                  []int64{100},
		CTRPPM:                []uint32{CTRPPMUnit},
		Reserves:              []int64{0},
		DailyBudgets:          []int64{0},
		PacingOpen:            []uint8{PacingOpen},
		DeviceMasks:           []uint8{1},
		CategoryMasks:         []uint64{1},
		GeoHashes:             []uint32{7},
		Weights:               []uint32{1},
		BoostPPM:              []uint32{CTRPPMUnit},
		BudgetIndices:         []uint32{99999},
		CustomerBudgetIndices: []uint32{invalidCustomerBudgetIdx},
	})
	buildGeoIndex(reg.LoadShard(7))

	_, reason, panicked := faultRunAuction(reg, stdReq(7, 50))
	assert.False(t, panicked)
	assert.Equal(t, NoBidCorruptCatalog, reason)
	faultproof.Log(t, "rtb_oob_budget_idx", map[string]string{"outcome": "corrupt_catalog", "no_debit": "true"})
}

func TestFault_NegativeBidInCatalog(t *testing.T) {
	store := NewBudgetStore()
	reg := NewRegistry(store)
	reg.UpdateCampaigns([]CampaignData{{
		ID: CampaignID(1), Bid: -50, DeviceMask: 1, CategoryMask: 1,
		GeoHashVal: 7, Budget: 1000,
	}})

	_, reason, panicked := faultRunAuction(reg, stdReq(7, 0))
	assert.False(t, panicked)
	assert.False(t, reason.OK())
	faultproof.Log(t, "rtb_negative_bid", map[string]string{"outcome": reason.String()})
}

func TestFault_ZeroCampaignBudget(t *testing.T) {
	store := NewBudgetStore()
	reg := NewRegistry(store)
	reg.UpdateCampaigns(singleCampaign(CampaignID(1), 100, 0))

	_, reason, panicked := faultRunAuction(reg, stdReq(7, 50))
	assert.False(t, panicked)
	assert.Equal(t, NoBidNoCandidates, reason)
	faultproof.Log(t, "rtb_zero_budget", map[string]string{"outcome": "no_candidates"})
}

func TestFault_CustomerRollbackOnSpendFail(t *testing.T) {
	store := NewBudgetStore()
	campIdx := store.GetOrAllocateSlot(CampaignID(1), 1000)
	custIdx := store.GetOrAllocateCustomerSlot(CustomerID(9), 50)

	beforeCamp, beforeCust, _ := faultBudgetTriple(store, campIdx, custIdx)
	ok := store.CheckAndSpendAll(campIdx, custIdx, 100, 0)
	assert.False(t, ok)

	afterCamp, afterCust, _ := faultBudgetTriple(store, campIdx, custIdx)
	assert.Equal(t, beforeCamp, afterCamp, "campaign budget must rollback")
	assert.Equal(t, beforeCust, afterCust, "customer budget must rollback")
	faultproof.Log(t, "rtb_customer_insufficient", map[string]string{"rollback": "true"})
}

func TestFault_DailyRollbackOnSpendFail(t *testing.T) {
	store := NewBudgetStore()
	campIdx := store.GetOrAllocateSlot(CampaignID(1), 1000)
	custIdx := store.GetOrAllocateCustomerSlot(CustomerID(9), 500)

	beforeCamp, beforeCust, beforeDaily := faultBudgetTriple(store, campIdx, custIdx)
	ok := store.CheckAndSpendAll(campIdx, custIdx, 100, 50)
	assert.False(t, ok)

	afterCamp, afterCust, afterDaily := faultBudgetTriple(store, campIdx, custIdx)
	assert.Equal(t, beforeCamp, afterCamp)
	assert.Equal(t, beforeCust, afterCust)
	assert.Equal(t, beforeDaily, afterDaily)
	faultproof.Log(t, "rtb_daily_cap_on_spend", map[string]string{"rollback": "all"})
}

func TestFault_SpendOutOfRangeIndex(t *testing.T) {
	store := NewBudgetStore()
	panicked := false
	func() {
		defer func() {
			if recover() != nil {
				panicked = true
			}
		}()
		_ = store.CheckAndSpendAll(99999, invalidCustomerBudgetIdx, 10, 0)
	}()
	assert.False(t, panicked)
	faultproof.Log(t, "rtb_oob_spend_idx", map[string]string{"no_panic": "true"})
}

func TestFault_ExactBudgetSingleWin(t *testing.T) {
	store := NewBudgetStore()
	reg := NewRegistry(store)
	reg.UpdateCampaigns(singleCampaign(CampaignID(1), 100, 100))

	res1, r1, _ := faultRunAuction(reg, stdReq(7, 50))
	require.True(t, r1.OK())
	assert.Equal(t, int64(50), res1.Price)
	assert.Equal(t, int64(50), store.GetBudget(CampaignID(1)))

	_, r2, panicked := faultRunAuction(reg, stdReq(7, 50))
	assert.False(t, panicked)
	assert.Equal(t, NoBidNoCandidates, r2)
	faultproof.Log(t, "rtb_exact_one_clearing", map[string]string{
		"first_price": strconv.FormatInt(res1.Price, 10),
		"second":      r2.String(),
	})
}

func TestFault_ClearingPriceNotAboveBid(t *testing.T) {
	store := NewBudgetStore()
	reg := NewRegistry(store)
	reg.UpdateCampaigns([]CampaignData{
		{ID: 1, Bid: 200, DeviceMask: 1, CategoryMask: 1, GeoHashVal: 7, Budget: 5000},
		{ID: 2, Bid: 150, DeviceMask: 1, CategoryMask: 1, GeoHashVal: 7, Budget: 5000},
	})

	res, reason, panicked := faultRunAuction(reg, stdReq(7, 50))
	require.False(t, panicked)
	require.True(t, reason.OK())
	assert.LessOrEqual(t, res.Price, int64(200))
	faultproof.Log(t, "rtb_clearing_price_cap", map[string]string{
		"clearing_price": strconv.FormatInt(res.Price, 10),
		"winner_bid":     "200",
	})
}

func TestFault_SetBudgetZeroRace(t *testing.T) {
	store := NewBudgetStore()
	reg := NewRegistry(store)
	cid := CampaignID(1)
	reg.UpdateCampaigns(singleCampaign(cid, 100, 200))

	var wg sync.WaitGroup
	stop := make(chan struct{})
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
				store.SetBudget(cid, 0)
			}
		}
	}()

	for range 200 {
		reg.RunAuction(stdReq(7, 50))
	}
	close(stop)
	wg.Wait()

	assert.GreaterOrEqual(t, store.GetBudget(cid), int64(0))
	faultproof.Log(t, "rtb_concurrent_zero_budget", map[string]string{"min_budget_gte": "0"})
}

func TestFault_AllPacingClosed(t *testing.T) {
	store := NewBudgetStore()
	reg := NewRegistry(store)
	reg.UpdateCampaigns([]CampaignData{{
		ID: CampaignID(1), Bid: 100, PacingOpen: PacingClosed,
		DeviceMask: 1, CategoryMask: 1, GeoHashVal: 7, Budget: 1000,
	}})

	_, reason, panicked := faultRunAuction(reg, stdReq(7, 50))
	assert.False(t, panicked)
	assert.Equal(t, NoBidPacingClosed, reason)
	faultproof.Log(t, "rtb_all_pacing_closed", nil)
}

func TestFault_PacingBeatsDaily(t *testing.T) {
	store := NewBudgetStore()
	reg := NewRegistry(store)
	reg.UpdateCampaigns([]CampaignData{
		{ID: 1, Bid: 100, PacingOpen: PacingClosed, DeviceMask: 1, CategoryMask: 1, GeoHashVal: 7, Budget: 1000},
		{ID: 2, Bid: 100, DailyBudget: 10, DeviceMask: 1, CategoryMask: 1, GeoHashVal: 7, Budget: 1000},
	})
	idx := reg.LoadShard(7).BudgetIndices[1]
	store.addDailySpendLocked(idx, 10)

	_, reason, panicked := faultRunAuction(reg, stdReq(7, 50))
	assert.False(t, panicked)
	assert.Equal(t, NoBidPacingClosed, reason)
	faultproof.Log(t, "rtb_pacing_over_daily", map[string]string{"priority": "pacing"})
}

func TestFault_DailyOnlyBlocked(t *testing.T) {
	store := NewBudgetStore()
	reg := NewRegistry(store)
	reg.UpdateCampaigns([]CampaignData{{
		ID: 1, Bid: 100, DailyBudget: 50, DeviceMask: 1, CategoryMask: 1, GeoHashVal: 7, Budget: 1000,
	}})
	idx := reg.LoadShard(7).BudgetIndices[0]
	store.addDailySpendLocked(idx, 60)

	_, reason, panicked := faultRunAuction(reg, stdReq(7, 50))
	assert.False(t, panicked)
	assert.Equal(t, NoBidDailyCapExceeded, reason)
	faultproof.Log(t, "rtb_daily_only_blocked", nil)
}

func TestFault_ReserveAboveSecondPrice(t *testing.T) {
	store := NewBudgetStore()
	reg := NewRegistry(store)
	reg.UpdateCampaigns([]CampaignData{
		{ID: 1, Bid: 200, Reserve: 180, DeviceMask: 1, CategoryMask: 1, GeoHashVal: 7, Budget: 5000},
		{ID: 2, Bid: 120, DeviceMask: 1, CategoryMask: 1, GeoHashVal: 7, Budget: 5000},
	})

	res, reason, panicked := faultRunAuction(reg, stdReq(7, 50))
	require.False(t, panicked)
	require.True(t, reason.OK())
	assert.Equal(t, int64(180), res.Price)
	faultproof.Log(t, "rtb_reserve_lifts_clearing", map[string]string{"price": "180"})
}

func TestFault_FirstPriceWithReserve(t *testing.T) {
	store := NewBudgetStore()
	reg := NewRegistry(store)
	reg.SetClearingMode(ClearingFirstPrice)
	reg.UpdateCampaigns([]CampaignData{{
		ID: 1, Bid: 200, Reserve: 150, DeviceMask: 1, CategoryMask: 1, GeoHashVal: 7, Budget: 5000,
	}})

	res, reason, panicked := faultRunAuction(reg, stdReq(7, 50))
	require.False(t, panicked)
	require.True(t, reason.OK())
	assert.Equal(t, int64(200), res.Price)
	faultproof.Log(t, "rtb_first_price_reserve", map[string]string{"winner_bid": "200"})
}

func TestFault_ZeroCTRPNormalized(t *testing.T) {
	store := NewBudgetStore()
	reg := NewRegistry(store)
	reg.UpdateCampaigns([]CampaignData{{
		ID: 1, Bid: 100, CTRPPM: 0, DeviceMask: 1, CategoryMask: 1, GeoHashVal: 7, Budget: 1000,
	}})

	_, reason, panicked := faultRunAuction(reg, stdReq(7, 50))
	assert.False(t, panicked)
	assert.True(t, reason.OK())
	faultproof.Log(t, "rtb_zero_ctr_normalized", map[string]string{"outcome": "ok"})
}

func TestFault_SharedCustomerDrains(t *testing.T) {
	store := NewBudgetStore()
	reg := NewRegistry(store)
	cust := CustomerID(42)
	reg.UpdateCampaigns([]CampaignData{
		{ID: 1, Bid: 100, CustomerID: cust, CustomerBudget: 120, DeviceMask: 1, CategoryMask: 1, GeoHashVal: 7, Budget: 5000},
		{ID: 2, Bid: 90, CustomerID: cust, CustomerBudget: 120, Weight: 2, DeviceMask: 1, CategoryMask: 1, GeoHashVal: 7, Budget: 5000},
	})

	_, r1, _ := faultRunAuction(reg, stdReq(7, 50))
	require.True(t, r1.OK())
	custIdx := reg.LoadShard(7).CustomerBudgetIndices[0]
	assert.Equal(t, int64(30), store.LoadCustomerBudget(custIdx))

	_, r2, panicked := faultRunAuction(reg, stdReq(7, 50))
	assert.False(t, panicked)
	assert.False(t, r2.OK())
	faultproof.Log(t, "rtb_shared_customer", map[string]string{"second_auction": "no_bid"})
}

func TestFault_ZeroCustomerIDDisabled(t *testing.T) {
	store := NewBudgetStore()
	reg := NewRegistry(store)
	reg.UpdateCampaigns([]CampaignData{{
		ID: 1, Bid: 100, CustomerID: 0, CustomerBudget: 10,
		DeviceMask: 1, CategoryMask: 1, GeoHashVal: 7, Budget: 1000,
	}})

	sh := reg.LoadShard(7)
	assert.Equal(t, invalidCustomerBudgetIdx, sh.CustomerBudgetIndices[0])

	_, reason, panicked := faultRunAuction(reg, stdReq(7, 50))
	assert.False(t, panicked)
	assert.True(t, reason.OK())
	faultproof.Log(t, "rtb_customer_id_zero", map[string]string{"ignores_customer_pool": "true"})
}

func TestFault_CustomerExhaustedNoCampaignDebit(t *testing.T) {
	store := NewBudgetStore()
	reg := NewRegistry(store)
	cust := CustomerID(7)
	reg.UpdateCampaigns([]CampaignData{{
		ID: 1, Bid: 100, CustomerID: cust, CustomerBudget: 40,
		DeviceMask: 1, CategoryMask: 1, GeoHashVal: 7, Budget: 1000,
	}})

	before := store.GetBudget(CampaignID(1))
	_, reason, panicked := faultRunAuction(reg, stdReq(7, 50))
	assert.False(t, panicked)
	assert.Equal(t, NoBidNoCandidates, reason)
	assert.Equal(t, before, store.GetBudget(CampaignID(1)))
	faultproof.Log(t, "rtb_customer_low_prefilter", map[string]string{"no_campaign_debit": "true"})
}

func TestFault_CatalogClearDuringAuction(t *testing.T) {
	store := NewBudgetStore()
	reg := NewRegistry(store)
	reg.UpdateCampaigns(singleCampaign(CampaignID(1), 100, 10_000))

	var panicked atomic.Bool
	var wg sync.WaitGroup
	stop := make(chan struct{})

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
				func() {
					defer func() {
						if recover() != nil {
							panicked.Store(true)
						}
					}()
					reg.RunAuction(stdReq(7, 50))
				}()
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for range 500 {
			reg.UpdateCampaigns(nil)
			reg.UpdateCampaigns(singleCampaign(CampaignID(1), 100, 10_000))
		}
		close(stop)
	}()

	wg.Wait()
	assert.False(t, panicked.Load())
	assert.GreaterOrEqual(t, store.GetBudget(CampaignID(1)), int64(0))
	faultproof.Log(t, "rtb_catalog_rebuild_during_auction", map[string]string{"no_panic": "true"})
}

func TestFault_ParallelDrainNonNegative(t *testing.T) {
	store := NewBudgetStore()
	reg := NewRegistry(store)
	reg.UpdateCampaigns(singleCampaign(CampaignID(1), 100, 100))

	var wg sync.WaitGroup
	for range 32 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 50 {
				reg.RunAuction(stdReq(7, 50))
			}
		}()
	}
	wg.Wait()

	assert.GreaterOrEqual(t, store.GetBudget(CampaignID(1)), int64(0))
	faultproof.Log(t, "rtb_parallel_drain", map[string]string{
		"remaining": strconv.FormatInt(store.GetBudget(CampaignID(1)), 10),
	})
}
