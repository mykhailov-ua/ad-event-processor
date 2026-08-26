package ingestion

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/ingestion/traceprobe"
	"ad-event-processor/internal/metrics"

	"github.com/google/uuid"
	redis "github.com/redis/go-redis/v9"
)

var (
	ErrRateLimitExceeded      = errors.New("rate limit exceeded")
	ErrDuplicateEvent         = errors.New("duplicate event detected")
	ErrBudgetExhausted        = errors.New("budget exhausted")
	ErrCampaignNotFound       = errors.New("campaign not found in registry")
	ErrPacingExhausted        = errors.New("pacing exhausted")
	ErrFreqLimitExceeded      = domain.ErrFreqLimitExceeded
	ErrGeoBlocked             = errors.New("geo-targeting blocked")
	ErrScheduleBlocked        = errors.New("outside delivery schedule")
	ErrFraudDetected          = errors.New("fraud detected")
	ErrEmergencyBreakerActive = domain.ErrEmergencyBreakerActive
	ErrBidFloorNotMet         = errors.New("bid floor not met")
	ErrMigrationFenced        = domain.ErrMigrationFenced
	ErrLicenseExpired         = errors.New("license expired")
	ErrDailyQuotaExceeded     = errors.New("daily quota exceeded")
	ErrRegistryStale          = errors.New("registry stale: campaign unknown while control plane unreachable")
	ErrShardUnavailable       = errors.New("shard unavailable")
	ErrFilterTimeout          = errors.New("filter timeout")
)

type bufWrapper struct {
	buf []byte
}

var bufPool = sync.Pool{
	New: func() any {
		return &bufWrapper{
			buf: make([]byte, 0, 128),
		}
	},
}

const hexChars = "0123456789abcdef"

func appendUUID(dst []byte, u uuid.UUID) []byte {
	return append(dst,
		hexChars[u[0]>>4], hexChars[u[0]&0xf],
		hexChars[u[1]>>4], hexChars[u[1]&0xf],
		hexChars[u[2]>>4], hexChars[u[2]&0xf],
		hexChars[u[3]>>4], hexChars[u[3]&0xf],
		'-',
		hexChars[u[4]>>4], hexChars[u[4]&0xf],
		hexChars[u[5]>>4], hexChars[u[5]&0xf],
		'-',
		hexChars[u[6]>>4], hexChars[u[6]&0xf],
		hexChars[u[7]>>4], hexChars[u[7]&0xf],
		'-',
		hexChars[u[8]>>4], hexChars[u[8]&0xf],
		hexChars[u[9]>>4], hexChars[u[9]&0xf],
		'-',
		hexChars[u[10]>>4], hexChars[u[10]&0xf],
		hexChars[u[11]>>4], hexChars[u[11]&0xf],
		hexChars[u[12]>>4], hexChars[u[12]&0xf],
		hexChars[u[13]>>4], hexChars[u[13]&0xf],
		hexChars[u[14]>>4], hexChars[u[14]&0xf],
		hexChars[u[15]>>4], hexChars[u[15]&0xf],
	)
}

func unsafeString(b []byte) string {
	if len(b) == 0 {
		return ""
	}

	return unsafe.String(&b[0], len(b))
}

type FraudFilter struct {
	geo        GeoProvider
	dcASN      *DCASNTable
	asnLookup  ASNLookup
	sampleSeq  atomic.Uint64
	sampleMask uint64
}

type ASNLookup interface {
	LookupASN(ip string) (uint32, bool)
}

func NewFraudFilter(geo GeoProvider) *FraudFilter {
	return &FraudFilter{
		geo: geo,
	}
}

func (f *FraudFilter) ConfigureDCASN(table *DCASNTable, lookup ASNLookup, sampleMaskCfg int) {
	if f == nil {
		return
	}
	f.dcASN = table
	f.asnLookup = lookup
	if sampleMaskCfg == 0 {
		f.sampleMask = dcASNCheckSampleMask
	} else {
		f.sampleMask = histogramSampleMaskFromConfig(sampleMaskCfg)
	}
}

func (f *FraudFilter) Check(ctx context.Context, evt *domain.Event) error {
	if evt == nil {
		return nil
	}
	isAnon, anonKnown := ingestAnonymousResolved(evt)
	if !anonKnown && f.geo != nil {
		var err error
		isAnon, err = f.geo.IsAnonymous(evt.IP)
		anonKnown = err == nil
	}
	if anonKnown && isAnon {
		addFraudSignal(evt, FraudReasonDatacenterIP)
		return nil
	}

	f.checkDCASN(evt, anonKnown && !isAnon)
	return nil
}

func ingestAnonymousResolved(evt *domain.Event) (isAnon bool, ok bool) {
	if evt == nil || !evt.IngestGeoResolved {
		return false, false
	}
	return evt.IngestAnonymous, true
}

const dcASNCheckSampleMask = 7

func (f *FraudFilter) checkDCASN(evt *domain.Event, force bool) {
	if f == nil || f.dcASN == nil || f.asnLookup == nil || !f.dcASN.Ready() || evt.IP == "" {
		return
	}
	if !force && !shouldSampleHistogram(f.sampleSeq.Add(1), f.sampleMask) {
		return
	}
	metrics.DCASNCheckTotal.Inc()
	asn, ok := f.asnLookup.LookupASN(evt.IP)
	if !ok {
		return
	}
	if f.dcASN.IsDatacenter(asn) {
		metrics.DCASNMatchTotal.Inc()
		addFraudSignal(evt, FraudReasonDatacenterIP)
	}
}

type GeoFilter struct {
	geo      GeoProvider
	registry domain.CampaignRegistry
}

func NewGeoFilter(geo GeoProvider, registry domain.CampaignRegistry) *GeoFilter {
	return &GeoFilter{
		geo:      geo,
		registry: registry,
	}
}

func (f *GeoFilter) Check(ctx context.Context, evt *domain.Event) error {
	start := monotonicNano()
	err := f.checkGeo(evt)
	observeHistogramSampled(&geoMetricsSeq, luaMetricsSampleMask, filterGeoDuration, start)
	return err
}

func (f *GeoFilter) checkGeo(evt *domain.Event) error {
	camp, ok := getCampaignFromEvent(f.registry, evt)
	if !ok {
		if reg, ok := f.registry.(*Registry); ok && reg.IsStaleMode() {
			return ErrRegistryStale
		}
		return ErrCampaignNotFound
	}

	if len(camp.TargetCountries) == 0 {
		return nil
	}

	var country string
	if evt.IngestGeoResolved {
		country = evt.GeoCountry
	} else {
		var err error
		country, err = f.geo.GetCountry(evt.IP)
		if err != nil {
			filterGeoLookupErrors.Inc()
			return nil
		}
	}
	if country == "" {
		filterGeoLookupErrors.Inc()
		return nil
	}

	if _, allowed := camp.TargetCountries[country]; allowed {
		return nil
	}

	return ErrGeoBlocked
}

type BudgetFilter struct {
	manager          domain.BudgetManager
	registry         domain.CampaignRegistry
	clickAmount      int64
	impressionAmount int64
}

func NewBudgetFilter(manager domain.BudgetManager, registry domain.CampaignRegistry, clickAmount, impressionAmount int64) *BudgetFilter {
	return &BudgetFilter{
		manager:          manager,
		registry:         registry,
		clickAmount:      clickAmount,
		impressionAmount: impressionAmount,
	}
}

func (f *BudgetFilter) Check(ctx context.Context, evt *domain.Event) error {
	customerID, ok := f.registry.GetCustomerID(evt.CampaignID)
	if !ok {
		return ErrCampaignNotFound
	}

	amount := f.clickAmount
	if evt.Type == "impression" {
		amount = f.impressionAmount
	}

	allowed, err := f.manager.CheckAndSpend(ctx, customerID, evt.CampaignID, evt.ClickID, amount)
	if err != nil {
		return err
	}
	if !allowed {
		return ErrBudgetExhausted
	}
	return nil
}

type EventFilter interface {
	Check(ctx context.Context, evt *domain.Event) error
}

type FilterEngine struct {
	filters  []EventFilter
	timeout  time.Duration
	registry domain.CampaignRegistry
	watcher  *SettingsWatcher
}

func NewFilterEngine(timeout time.Duration, filters ...EventFilter) *FilterEngine {
	return &FilterEngine{filters: filters, timeout: timeout}
}

func (e *FilterEngine) SetRegistry(registry domain.CampaignRegistry) {
	e.registry = registry
}

func (e *FilterEngine) SetSettingsWatcher(watcher *SettingsWatcher) {
	e.watcher = watcher
}

func (e *FilterEngine) SetDeferStreamToProducer(deferWrite bool) {
	if e == nil {
		return
	}
	for _, filter := range e.filters {
		if uf, ok := filter.(*UnifiedFilter); ok {
			uf.SetDeferStreamToProducer(deferWrite)
		}
	}
}

func (e *FilterEngine) RollbackDebit(ctx context.Context, evt *domain.Event, registry domain.CampaignRegistry) {
	if e == nil || evt == nil || registry == nil {
		return
	}
	campInfo, ok := getCampaignFromEvent(registry, evt)
	if !ok {
		return
	}
	for _, f := range e.filters {
		if uf, ok := f.(*UnifiedFilter); ok {
			debitAmount := uf.clickAmountMicro
			if evt.Type == "impression" {
				debitAmount = uf.impressionAmountMicro
			}
			isLocalQuanta := uf.localQuantaFullSkipEligible(evt, campInfo)
			uf.RollbackDebit(ctx, evt, campInfo, debitAmount, isLocalQuanta)
		}
	}
}

func (e *FilterEngine) Timeout() time.Duration {
	if e == nil {
		return 0
	}
	return e.timeout
}

func (e *FilterEngine) Check(ctx context.Context, evt *domain.Event) error {
	slot := uint32(0)
	if evt != nil {
		slot = uint32(CampaignSlotIndex(evt.CampaignID))
	}
	traceprobe.FilterCheckEnter(slot)
	err := e.checkInner(ctx, evt)
	traceprobe.FilterCheckExit(slot)
	return err
}

func (e *FilterEngine) checkInner(ctx context.Context, evt *domain.Event) error {
	if e.timeout > 0 && evt != nil {
		evt.FilterDeadlineMono = monotonicNano() + e.timeout.Nanoseconds()
	}
	acc := attachFraudAccumulator(evt)

	var boost uint8
	if e.watcher != nil && evt != nil {
		boosts := e.watcher.GetFraudScoreBoosts()
		if boosts != nil {
			boost = boosts.Boosts[evt.CampaignID]
		}
	}

	var retErr error
	for _, f := range e.filters {
		if f == nil {
			continue
		}
		if filterDeadlineExceededEvt(evt, ctx) {
			retErr = ErrFilterTimeout
			break
		}
		if uf, ok := f.(*UnifiedFilter); ok && acc != nil && acc.count > 0 {
			var camp *domain.Campaign
			if e.registry != nil && evt != nil {
				camp, _ = getCampaignFromEvent(e.registry, evt)
			}
			if acc.shouldShortCircuitFraudBudget() {
				layer, err := applyFraudLayerDecision(evt, acc, camp, boost)
				if err != nil {
					retErr = err
					break
				}
				if layer == FraudLayerL1Reject {
					retErr = ErrFraudDetected
				}
				break
			}
			tier := applyFraudAccumulatorForCampaign(evt, acc, camp)
			if decideFraudLayer(acc, tier) == FraudLayerL2Shadow {
				prevSkip := uf.skipBudgetDebitAny
				uf.skipBudgetDebitAny = oneAny
				if err := uf.Check(ctx, evt); err != nil {
					uf.skipBudgetDebitAny = prevSkip
					retErr = err
					break
				}
				uf.skipBudgetDebitAny = prevSkip
				break
			}
			continue
		}
		if err := f.Check(ctx, evt); err != nil {
			retErr = err
			break
		}
	}

	if retErr == nil {
		var camp *domain.Campaign
		if e.registry != nil && evt != nil {
			camp, _ = getCampaignFromEvent(e.registry, evt)
		}
		layer, err := applyFraudLayerDecision(evt, acc, camp, boost)
		if err != nil {
			retErr = err
		} else if layer == FraudLayerL1Reject {
			retErr = ErrFraudDetected
		}
	}

	if evt != nil && evt.FilterDeadlineMono > 0 {
		evt.FilterDeadlineMono = 0
	}
	releaseFraudAccumulator(evt, acc)
	return retErr
}

type DuplicateEventFilter struct {
	redisClient redis.Cmdable
	ttl         time.Duration
}

func NewDuplicateEventFilter(redisClient redis.Cmdable, ttl time.Duration) *DuplicateEventFilter {
	return &DuplicateEventFilter{
		redisClient: redisClient,
		ttl:         ttl,
	}
}

func (f *DuplicateEventFilter) Check(ctx context.Context, evt *domain.Event) error {
	if evt.ClickID == "" {
		return nil
	}

	w := bufPool.Get().(*bufWrapper)
	w.buf = w.buf[:0]
	w.buf = append(w.buf, "dup:"...)
	w.buf = append(w.buf, evt.Type...)
	w.buf = append(w.buf, ':')
	w.buf = append(w.buf, evt.ClickID...)
	key := unsafeString(w.buf)

	ok, err := f.redisClient.SetNX(ctx, key, "1", f.ttl).Result()
	bufPool.Put(w)

	if err != nil {
		return err
	}

	if !ok {
		return ErrDuplicateEvent
	}

	return nil
}

type EmergencyBreakerFilter struct {
	watcher *SettingsWatcher
}

func NewEmergencyBreakerFilter(watcher *SettingsWatcher) *EmergencyBreakerFilter {
	return &EmergencyBreakerFilter{watcher: watcher}
}

func (f *EmergencyBreakerFilter) Check(ctx context.Context, evt *domain.Event) error {
	if f.watcher != nil && f.watcher.Get().EmergencyBreaker {
		return ErrEmergencyBreakerActive
	}
	return nil
}

type placementCacheItem struct {
	blacklisted bool
	expiry      int64
}

const placementCacheShards = 128

// placementBlacklistCacheTTL bounds stale blacklist visibility without per-event HEXISTS.
const placementBlacklistCacheTTL = 5 * time.Second

const placementCacheMaxEntriesPerShard = 2048

type placementCacheKey struct {
	campaignID uuid.UUID
	placement  string
}

type placementCacheShard struct {
	snap atomic.Pointer[placementShardSnapshot]
}

type placementShardSnapshot struct {
	entries map[placementCacheKey]placementCacheItem
}

type PlacementBlacklistFilter struct {
	redisShards []redis.UniversalClient
	sharder     Sharder
	shards      [placementCacheShards]placementCacheShard
}

func NewPlacementBlacklistFilter(redisShards []redis.UniversalClient) *PlacementBlacklistFilter {
	f := &PlacementBlacklistFilter{redisShards: redisShards}
	for i := range placementCacheShards {
		f.shards[i].snap.Store(&placementShardSnapshot{
			entries: make(map[placementCacheKey]placementCacheItem, 64),
		})
	}
	return f
}

func (f *PlacementBlacklistFilter) SetSharder(sharder Sharder) {
	if f != nil {
		f.sharder = sharder
	}
}

func placementShardStore(shard *placementCacheShard, key placementCacheKey, item placementCacheItem, nowMs int64) {
	for {
		old := shard.snap.Load()
		next := placementCloneEntries(old, nowMs, key, item)
		newSnap := &placementShardSnapshot{entries: next}
		if shard.snap.CompareAndSwap(old, newSnap) {
			return
		}
	}
}

func placementCloneEntries(old *placementShardSnapshot, nowMs int64, key placementCacheKey, item placementCacheItem) map[placementCacheKey]placementCacheItem {
	var oldMap map[placementCacheKey]placementCacheItem
	if old != nil {
		oldMap = old.entries
	}
	next := make(map[placementCacheKey]placementCacheItem, len(oldMap)+1)
	for k, v := range oldMap {
		if nowMs < v.expiry {
			next[k] = v
		}
	}
	if len(next) >= placementCacheMaxEntriesPerShard {
		placementCachePruneMap(next, nowMs)
	}
	next[key] = item
	return next
}

func placementCachePruneMap(entries map[placementCacheKey]placementCacheItem, now int64) {
	for k, v := range entries {
		if now >= v.expiry {
			delete(entries, k)
		}
	}
	for len(entries) >= placementCacheMaxEntriesPerShard {
		for k := range entries {
			delete(entries, k)
			break
		}
	}
}

func (f *PlacementBlacklistFilter) Check(ctx context.Context, evt *domain.Event) error {
	if evt == nil || evt.PlacementID == "" {
		return nil
	}

	key := placementCacheKey{
		campaignID: evt.CampaignID,
		placement:  evt.PlacementID,
	}

	h := uint32(evt.CampaignID[0]) | (uint32(evt.CampaignID[1]) << 8)
	shardIdx := h % placementCacheShards
	shard := &f.shards[shardIdx]

	nowMs := cachedUnixMilliNow()
	snap := shard.snap.Load()
	if snap != nil {
		if item, ok := snap.entries[key]; ok && nowMs < item.expiry {
			if item.blacklisted {
				return ErrPlacementBlocked
			}
			return nil
		}
	}

	redisClient := pickGlobalReadShardForCampaign(f.redisShards, f.sharder, evt.CampaignID)
	if redisClient == nil {
		return nil
	}

	w := bufPool.Get().(*bufWrapper)
	w.buf = appendCampaignHashTag(w.buf[:0], evt.CampaignID)
	w.buf = append(w.buf, "blacklist:placement:"...)
	w.buf = appendUUID(w.buf, evt.CampaignID)
	redisKey := unsafeString(w.buf)

	isBlacklisted, err := redisClient.HExists(ctx, redisKey, evt.PlacementID).Result()
	bufPool.Put(w)
	if err != nil {
		return nil
	}

	placementShardStore(shard, key, placementCacheItem{
		blacklisted: isBlacklisted,
		expiry:      nowMs + placementBlacklistCacheTTL.Milliseconds(),
	}, nowMs)

	if isBlacklisted {
		return ErrPlacementBlocked
	}
	return nil
}

var ErrPlacementBlocked = errors.New("placement blocked")
