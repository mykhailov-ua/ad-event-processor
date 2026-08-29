package rtb

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/metrics"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type RtbBudgetReconcileConfig struct {
	Interval            time.Duration
	DivergenceThreshold int64
	SampleSize          int
}

type RtbBudgetReconcileWorker struct {
	cfg         RtbBudgetReconcileConfig
	registry    CampaignSource
	catalog     *RtbCatalog
	redisShards []redis.UniversalClient
	sharder     domain.Sharder
	cancel      context.CancelFunc
	wg          sync.WaitGroup
}

func NewRtbBudgetReconcileWorker(
	cfg RtbBudgetReconcileConfig,
	registry CampaignSource,
	catalog *RtbCatalog,
	redisShards []redis.UniversalClient,
	sharder domain.Sharder,
) *RtbBudgetReconcileWorker {
	if cfg.Interval <= 0 {
		cfg.Interval = 30 * time.Second
	}
	if cfg.DivergenceThreshold <= 0 {
		cfg.DivergenceThreshold = 1000
	}
	if cfg.SampleSize <= 0 {
		cfg.SampleSize = 32
	}
	return &RtbBudgetReconcileWorker{
		cfg:         cfg,
		registry:    registry,
		catalog:     catalog,
		redisShards: redisShards,
		sharder:     sharder,
	}
}

func (w *RtbBudgetReconcileWorker) Start(ctx context.Context) {
	if w == nil || w.registry == nil || w.catalog == nil || len(w.redisShards) == 0 || w.sharder == nil {
		return
	}
	runCtx, cancel := context.WithCancel(ctx)
	w.cancel = cancel
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		w.run(runCtx)
	}()
}

func (w *RtbBudgetReconcileWorker) Close() {
	if w != nil && w.cancel != nil {
		w.cancel()
	}
}

func (w *RtbBudgetReconcileWorker) Wait(ctx context.Context) error {
	if w == nil {
		return nil
	}
	done := make(chan struct{})
	go func() {
		w.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (w *RtbBudgetReconcileWorker) run(ctx context.Context) {
	ticker := time.NewTicker(w.cfg.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.sample(ctx)
		}
	}
}

func (w *RtbBudgetReconcileWorker) sample(ctx context.Context) {
	sampleCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	campaigns := w.registry.ActiveCampaigns()
	if len(campaigns) == 0 {
		metrics.RtbBudgetReconcileHigh.Set(0)
		return
	}

	store := w.catalog.Registry().Store()

	limit := w.cfg.SampleSize
	if limit > len(campaigns) {
		limit = len(campaigns)
	}

	var maxDelta int64
	for i := range limit {
		camp := campaigns[i]
		if camp == nil || camp.BudgetCampaignKey == "" {
			continue
		}
		redisRem, ok := loadRedisCampaignBudget(sampleCtx, w.redisShards, w.sharder, camp)
		if !ok {
			continue
		}
		rtbRem := store.GetBudget(CampaignIDFromUUID(camp.ID))
		delta := redisRem - rtbRem
		if delta < 0 {
			delta = -delta
		}
		metrics.RtbBudgetReconcileDivergenceMicro.Observe(float64(delta))
		metrics.RtbBudgetReconcileSamplesTotal.Inc()
		if delta > maxDelta {
			maxDelta = delta
		}
	}

	if maxDelta > w.cfg.DivergenceThreshold {
		metrics.RtbBudgetReconcileHigh.Set(1)
		slog.Warn("rtb budget reconcile divergence high", "max_delta_micro", maxDelta, "threshold", w.cfg.DivergenceThreshold)
	} else {
		metrics.RtbBudgetReconcileHigh.Set(0)
	}
}

func ReconcileCampaignBudget(
	ctx context.Context,
	store *BudgetStore,
	redisShards []redis.UniversalClient,
	sharder domain.Sharder,
	camp *domain.Campaign,
) (redisRem int64, rtbRem int64, ok bool) {
	if store == nil || camp == nil || len(redisShards) == 0 || sharder == nil {
		return 0, 0, false
	}
	redisRem, ok = loadRedisCampaignBudget(ctx, redisShards, sharder, camp)
	if !ok {
		return 0, 0, false
	}
	rtbRem = store.GetBudget(CampaignIDFromUUID(camp.ID))
	return redisRem, rtbRem, true
}

type RtbBudgetSync struct {
	Authority BudgetAuthority
	Redis     []redis.UniversalClient
	Sharder   domain.Sharder
}

func SyncRTBBudgetState(
	ctx context.Context,
	store *BudgetStore,
	campaigns []*domain.Campaign,
	customerPools map[uuid.UUID]int64,
	sync RtbBudgetSync,
) {
	if store == nil || len(campaigns) == 0 {
		return
	}
	readRedis := sync.Authority == BudgetAuthorityRTB && len(sync.Redis) > 0 && sync.Sharder != nil

	for _, camp := range campaigns {
		if camp == nil {
			continue
		}
		campID := CampaignIDFromUUID(camp.ID)
		remaining := remainingBudgetMicro(camp)
		if readRedis && camp.BudgetCampaignKey != "" {
			if redisRem, ok := loadRedisCampaignBudget(ctx, sync.Redis, sync.Sharder, camp); ok {
				remaining = redisRem
			}
		}
		store.SetBudget(campID, remaining)

		if readRedis && camp.DailySpendKeyPrefix != "" {
			if spent, ok := loadRedisDailySpend(ctx, sync.Redis, sync.Sharder, camp); ok {
				if idx, exists := store.CampaignSlot(campID); exists {
					store.SetDailySpend(idx, spent)
				}
			}
		}
	}

	for customerID, pool := range customerPools {
		if pool < 0 {
			pool = 0
		}
		store.SetCustomerBudget(CustomerIDFromCustomerUUID(customerID), pool)
	}
}

func loadRedisCampaignBudget(
	ctx context.Context,
	redisShards []redis.UniversalClient,
	sharder domain.Sharder,
	camp *domain.Campaign,
) (int64, bool) {
	shard := sharder.GetShard(camp.ID)
	if shard < 0 || shard >= len(redisShards) {
		return 0, false
	}
	val, err := redisShards[shard].Get(ctx, camp.BudgetCampaignKey).Int64()
	if err != nil {
		return 0, false
	}
	if val < 0 {
		val = 0
	}
	return val, true
}

func loadRedisDailySpend(
	ctx context.Context,
	redisShards []redis.UniversalClient,
	sharder domain.Sharder,
	camp *domain.Campaign,
) (int64, bool) {
	shard := sharder.GetShard(camp.ID)
	if shard < 0 || shard >= len(redisShards) {
		return 0, false
	}
	loc := camp.Location
	if loc == nil {
		loc = time.UTC
	}
	keyBuf := make([]byte, 0, len(camp.DailySpendKeyPrefix)+8)
	keyBuf = append(keyBuf, camp.DailySpendKeyPrefix...)
	keyBuf = appendDate(keyBuf, time.Now().In(loc))
	key := string(keyBuf)

	val, err := redisShards[shard].Get(ctx, key).Int64()
	if err != nil {
		return 0, false
	}
	if val < 0 {
		val = 0
	}
	return val, true
}

type RtbCatalog struct {
	registry   *Registry
	dealIndex  *DealIndex
	DealFloors *DealFloorCache
	authority  BudgetAuthority
	winnerUUID atomic.Pointer[map[CampaignID]uuid.UUID]

	prebidIVT       atomic.Bool
	schainAllow     atomic.Pointer[SupplyChainAllowlistSnapshot]
	settingsWatcher FcapSnapshotProvider
	ingestGeo       GeoAnonLookup
}

func NewRtbCatalog(store *BudgetStore, authority BudgetAuthority) *RtbCatalog {
	return &RtbCatalog{
		registry:  NewRegistry(store),
		dealIndex: NewDealIndex(),
		authority: authority,
	}
}

func (c *RtbCatalog) Registry() *Registry {
	return c.registry
}

func (c *RtbCatalog) Authority() BudgetAuthority {
	return c.authority
}

func (c *RtbCatalog) SetAuthority(authority BudgetAuthority) {
	c.authority = authority
}

func (c *RtbCatalog) SetPrebidIVT(enabled bool) {
	c.prebidIVT.Store(enabled)
}

func (c *RtbCatalog) SetSupplyChainAllowlist(snap *SupplyChainAllowlistSnapshot) {
	if snap == nil {
		c.schainAllow.Store(nil)
		return
	}
	c.schainAllow.Store(snap)
}

func (c *RtbCatalog) ConfigureRtbGates(watcher FcapSnapshotProvider, geo GeoAnonLookup) {
	if c == nil {
		return
	}
	c.settingsWatcher = watcher
	c.ingestGeo = geo
}

func (c *RtbCatalog) SetDealFloors(cache *DealFloorCache) {
	c.DealFloors = cache
}

func (c *RtbCatalog) SyncActiveCampaigns(campaigns []*domain.Campaign, inputs map[uuid.UUID]RtbCampaignInput) {
	rows := BuildRtbCatalogRows(campaigns, inputs)
	c.registry.UpdateCampaigns(rows)
	c.rebuildWinnerUUID(rows, campaigns)
}

func (c *RtbCatalog) rebuildWinnerUUID(rows []CampaignData, campaigns []*domain.Campaign) {
	if len(rows) == 0 {
		empty := make(map[CampaignID]uuid.UUID)
		c.winnerUUID.Store(&empty)
		return
	}
	m := make(map[CampaignID]uuid.UUID, len(rows))
	for _, camp := range campaigns {
		if camp == nil {
			continue
		}
		m[CampaignIDFromUUID(camp.ID)] = camp.ID
	}
	c.winnerUUID.Store(&m)
}

func (c *RtbCatalog) LookupCreativeADM(geoHash uint32, campaignID CampaignID, creativeID CreativeID) ([]byte, uint8, bool) {
	if c == nil || c.registry == nil {
		return nil, 0, false
	}
	return c.registry.LookupCreativeWire(geoHash, campaignID, creativeID)
}

func (c *RtbCatalog) UUIDForWinner(id CampaignID) (uuid.UUID, bool) {
	ptr := c.winnerUUID.Load()
	if ptr == nil {
		return uuid.Nil, false
	}
	uid, ok := (*ptr)[id]
	return uid, ok
}

func (c *RtbCatalog) SyncCampaignRows(campaigns []*domain.Campaign, rows []CampaignData) {
	c.registry.UpdateCampaigns(rows)
	c.rebuildWinnerUUID(rows, campaigns)
}

func (c *RtbCatalog) SyncFromRegistry(src CampaignSource, inputs map[uuid.UUID]RtbCampaignInput) {
	if src == nil {
		c.registry.UpdateCampaigns(nil)
		return
	}
	c.SyncActiveCampaigns(src.ActiveCampaigns(), inputs)
}

func (c *RtbCatalog) SetClearingMode(mode ClearingMode) {
	c.registry.SetClearingMode(mode)
}

func (c *RtbCatalog) UpdateDeals(deals []DealData) {
	if c.dealIndex == nil {
		c.dealIndex = NewDealIndex()
	}
	c.dealIndex.UpdateDeals(deals)
}

func (c *RtbCatalog) DealCount() int {
	if c.dealIndex == nil {
		return 0
	}
	return c.dealIndex.Len()
}

func (c *RtbCatalog) LookupDeal(dealID string) (DealData, bool) {
	if c.dealIndex == nil {
		return DealData{}, false
	}
	return c.dealIndex.Lookup(dealID)
}

func (c *RtbCatalog) AllDeals() []DealData {
	if c.dealIndex == nil {
		return nil
	}
	return c.dealIndex.All()
}

func (c *RtbCatalog) EvaluateAuction(evt *domain.Event, targeting RtbTargetingInput) (AuctionResult, NoBidReason) {
	if c == nil || c.registry == nil {
		return AuctionResult{}, NoBidInvalidRequest
	}
	if reason := rtbPrefilterReject(c.settingsWatcher, c, targeting); reason != NoBidNone {
		return AuctionResult{}, reason
	}
	targeting = c.enrichTargetingDeal(targeting)
	if c.settingsWatcher != nil {
		c.registry.SetFcapSnapshot(c.settingsWatcher.GetFcapRtbSnapshot())
	}
	req := BidRequestFromEvent(evt, targeting)
	return c.registry.RunAuctionEval(&req)
}

func (c *RtbCatalog) RunAuction(evt *domain.Event, targeting RtbTargetingInput) (AuctionResult, NoBidReason) {
	if c == nil || c.registry == nil {
		return AuctionResult{}, NoBidInvalidRequest
	}
	if c.authority != BudgetAuthorityShadow {
		if reason := rtbPrefilterReject(c.settingsWatcher, c, targeting); reason != NoBidNone {
			return AuctionResult{}, reason
		}
		if c.prebidIVT.Load() {
			if reason := rtbPrebidIVTReject(true, c.ingestGeo, evt); reason != NoBidNone {
				return AuctionResult{}, reason
			}
		}
		if targeting.SchainCount > 0 {
			allow := c.schainAllow.Load()
			if allow != nil && !ValidateSchainNodes(targeting.Schain, allow) {
				return AuctionResult{}, NoBidSchainInvalid
			}
		}
	}
	targeting = c.enrichTargetingDeal(targeting)
	if c.settingsWatcher != nil {
		c.registry.SetFcapSnapshot(c.settingsWatcher.GetFcapRtbSnapshot())
	}
	req := BidRequestFromEvent(evt, targeting)
	if c.authority == BudgetAuthorityShadow {
		return c.registry.RunAuctionEval(&req)
	}
	res, reason := c.registry.RunAuction(&req)
	if reason.OK() && evt != nil {
		evt.ClearingPriceMicro = res.Price
	}
	return res, reason
}

func (c *RtbCatalog) enrichTargetingDeal(targeting RtbTargetingInput) RtbTargetingInput {
	if c == nil || c.dealIndex == nil {
		return targeting
	}
	var deal DealData
	var ok bool
	if targeting.DealIDLen > 0 {
		deal, ok = c.dealIndex.LookupBytes(targeting.DealIDBuf[:targeting.DealIDLen])
	}
	if !ok {
		return targeting
	}
	if deal.PacingOpen == PacingClosed {
		targeting.DealBlock = NoBidPacingClosed
		return targeting
	}
	geoBit := GeoBitFromHash(targeting.GeoHash)
	if (deal.GeoMask&geoBit) == 0 || (deal.CatMask&targeting.CategoryMask) == 0 {
		targeting.DealBlock = NoBidDealMismatch
		return targeting
	}
	if deal.Seats > 0 && int32(targeting.SeatCount) < deal.Seats {
		targeting.DealBlock = NoBidDealMismatch
		return targeting
	}
	return targeting
}
