package filter

import (
	"context"
	"hash/fnv"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/metrics"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type FraudFilter struct {
	geo        GeoProvider
	dcASN      *DCASNTable
	asnLookup  ASNLookup
	sampleSeq  atomic.Uint64
	sampleMask uint64
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
		f.sampleMask = HistogramSampleMaskFromConfig(sampleMaskCfg)
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
	if !force && !ShouldSampleHistogram(f.sampleSeq.Add(1), f.sampleMask) {
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
	geo                  GeoProvider
	registry             domain.CampaignRegistry
	acceptLangGeoEnabled atomic.Bool
}

func NewGeoFilter(geo GeoProvider, registry domain.CampaignRegistry) *GeoFilter {
	f := &GeoFilter{
		geo:      geo,
		registry: registry,
	}
	return f
}

func (f *GeoFilter) SetAcceptLangGeoEnabled(enabled bool) {
	f.acceptLangGeoEnabled.Store(enabled)
}

func (f *GeoFilter) Check(ctx context.Context, evt *domain.Event) error {
	start := MonotonicNano()
	err := f.checkGeo(evt)
	ObserveHistogramSampled(&geoMetricsSeq, luaMetricsSampleMask, filterGeoDuration, start)
	if err != nil {
		return err
	}
	f.checkAcceptLangGeo(evt)
	return nil
}

func (f *GeoFilter) checkAcceptLangGeo(evt *domain.Event) {
	if f == nil || evt == nil || !f.acceptLangGeoEnabled.Load() {
		return
	}
	camp, ok := GetCampaignFromEvent(f.registry, evt)
	if !ok || !camp.AcceptLangGeoEnabled {
		return
	}
	EnsureIngestGeo(f.geo, evt)
	if evt.AcceptLang == "" || evt.GeoCountry == "" {
		return
	}
	if AcceptLangGeoMismatch(evt.AcceptLang, evt.GeoCountry) {
		addFraudSignal(evt, FraudReasonAcceptLangGeoMismatch)
	}
}

func (f *GeoFilter) checkGeo(evt *domain.Event) error {
	camp, ok := GetCampaignFromEvent(f.registry, evt)
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

func (e *FilterEngine) StreamDeferredToProducer() bool {
	if e == nil {
		return false
	}
	for _, f := range e.filters {
		if df, ok := f.(DebitFilter); ok && df.StreamDeferredToProducer() {
			return true
		}
	}
	return false
}

func (e *FilterEngine) SetDeferStreamToProducer(deferWrite bool) {
	if e == nil {
		return
	}
	for _, f := range e.filters {
		if df, ok := f.(DebitFilter); ok {
			df.SetDeferStreamToProducer(deferWrite)
		}
	}
}

func (e *FilterEngine) RollbackDebit(ctx context.Context, evt *domain.Event, registry domain.CampaignRegistry) {
	if e == nil || evt == nil || registry == nil {
		return
	}
	campInfo, ok := GetCampaignFromEvent(registry, evt)
	if !ok {
		return
	}
	for _, f := range e.filters {
		if df, ok := f.(DebitFilter); ok {
			debitAmount := df.ClickAmountMicro()
			if evt.Type == "impression" {
				debitAmount = df.ImpressionAmountMicro()
			}
			isLocalQuanta := df.LocalQuantaFullSkipEligible(evt, campInfo)
			df.RollbackDebit(ctx, evt, campInfo, debitAmount, isLocalQuanta)
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
	FilterCheckEnter(slot)
	err := e.checkInner(ctx, evt)
	FilterCheckExit(slot)
	return err
}

func (e *FilterEngine) checkInner(ctx context.Context, evt *domain.Event) error {
	if e.timeout > 0 && evt != nil {
		evt.FilterDeadlineMono = MonotonicNano() + e.timeout.Nanoseconds()
	}
	acc := AttachFraudAccumulator(evt)

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
		if df, ok := f.(DebitFilter); ok && acc != nil && acc.count > 0 {
			var camp *domain.Campaign
			if e.registry != nil && evt != nil {
				camp, _ = GetCampaignFromEvent(e.registry, evt)
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
				df.SetSkipBudgetDebit(true)
				if err := df.Check(ctx, evt); err != nil {
					df.SetSkipBudgetDebit(false)
					retErr = err
					break
				}
				df.SetSkipBudgetDebit(false)
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
			camp, _ = GetCampaignFromEvent(e.registry, evt)
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
	ReleaseFraudAccumulator(evt, acc)
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

	w := bufPool.Get().(*BufWrapper)
	w.Buf = w.Buf[:0]
	w.Buf = append(w.Buf, "dup:"...)
	w.Buf = append(w.Buf, evt.Type...)
	w.Buf = append(w.Buf, ':')
	w.Buf = append(w.Buf, evt.ClickID...)
	key := UnsafeString(w.Buf)

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

	nowMs := CachedUnixMilliNow()
	snap := shard.snap.Load()
	if snap != nil {
		if item, ok := snap.entries[key]; ok && nowMs < item.expiry {
			if item.blacklisted {
				return ErrPlacementBlocked
			}
			return nil
		}
	}

	redisClient := PickGlobalReadShardForCampaign(f.redisShards, f.sharder, evt.CampaignID)
	if redisClient == nil {
		return nil
	}

	w := bufPool.Get().(*BufWrapper)
	w.Buf = appendCampaignHashTag(w.Buf[:0], evt.CampaignID)
	w.Buf = append(w.Buf, "blacklist:placement:"...)
	w.Buf = AppendUUID(w.Buf, evt.CampaignID)
	redisKey := UnsafeString(w.Buf)

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

type filterDeadlineKey struct{}

const maxFraudSignals = 4

const fraudAccumulatorMagic = 0x46524155445f01

type FraudTier uint8

const (
	FraudTierPass FraudTier = iota
	FraudTierSuspect
	FraudTierIVT
	FraudTierBlock
)

type fraudAccumulator struct {
	magic        uintptr
	score        uint32
	signals      [maxFraudSignals]FraudReasonID
	count        uint8
	boostApplied bool
}

var fraudAccPool = sync.Pool{
	New: func() any {
		return &fraudAccumulator{}
	},
}

func (a *fraudAccumulator) reset() {
	a.magic = fraudAccumulatorMagic
	a.score = 0
	a.count = 0
	a.boostApplied = false
}

func (a *fraudAccumulator) has(id FraudReasonID) bool {
	for i := uint8(0); i < a.count; i++ {
		if a.signals[i] == id {
			return true
		}
	}
	return false
}

func (a *fraudAccumulator) add(id FraudReasonID) {
	if id == FraudReasonNone || id >= fraudReasonCount || a.has(id) {
		return
	}
	weight := FraudSignalWeight(id)
	if weight == 0 {
		return
	}
	if a.count >= maxFraudSignals {
		return
	}
	a.signals[a.count] = id
	a.count++
	sum := a.score + uint32(weight)
	if sum > 100 {
		sum = 100
	}
	a.score = sum
}

func (a *fraudAccumulator) countFlags(want uint8) uint8 {
	if a == nil || want == 0 {
		return 0
	}
	var n uint8
	for i := uint8(0); i < a.count; i++ {
		if FraudSignalFlags(a.signals[i])&want != 0 {
			n++
		}
	}
	return n
}

func (a *fraudAccumulator) hasFlags(want uint8) bool {
	return a.countFlags(want) > 0
}

func (a *fraudAccumulator) shouldShortCircuitFraudBudget() bool {
	if a == nil || a.count == 0 {
		return false
	}
	if a.hasFlags(FraudSignalL3) {
		return true
	}
	return a.countFlags(FraudSignalL1High) >= 2
}

func eventHasFraudL3(evt *domain.Event) bool {
	acc, ok := fraudAccFromEvent(evt)
	if !ok {
		return false
	}
	return acc.hasFlags(FraudSignalL3)
}

func attachFilterDeadline(ctx context.Context, timeout time.Duration) context.Context {
	if timeout <= 0 {
		return ctx
	}
	deadlineMono := MonotonicNano() + timeout.Nanoseconds()
	return context.WithValue(ctx, filterDeadlineKey{}, deadlineMono)
}

func setFilterDeadlineOnEvent(evt *domain.Event, timeout time.Duration) {
	if evt != nil && timeout > 0 {
		evt.FilterDeadlineMono = MonotonicNano() + timeout.Nanoseconds()
	}
}

const openRTBScratchMagicFilter = 0x4f525442335f01

var openRTBScratchReleaser func(evt *domain.Event)

func SetOpenRTBScratchReleaser(fn func(evt *domain.Event)) {
	openRTBScratchReleaser = fn
}

func scratchIsOpenRTB(p unsafe.Pointer) bool {
	if p == nil {
		return false
	}
	return (*struct{ magic uint64 })(p).magic == openRTBScratchMagicFilter
}

func releaseOpenRTB3Scratch(evt *domain.Event) {
	if openRTBScratchReleaser != nil {
		openRTBScratchReleaser(evt)
		return
	}
	if evt != nil {
		evt.Scratch = nil
	}
}

func AttachFraudAccumulator(evt *domain.Event) *fraudAccumulator {
	if evt != nil && evt.Scratch != nil {
		if scratchIsOpenRTB(evt.Scratch) {
			releaseOpenRTB3Scratch(evt)
		} else if acc, ok := fraudAccFromEvent(evt); ok {
			return acc
		}
	}
	acc := fraudAccPool.Get().(*fraudAccumulator)
	acc.reset()
	if evt != nil {
		evt.Scratch = unsafe.Pointer(acc)
	}
	return acc
}

func ReleaseFraudAccumulator(evt *domain.Event, acc *fraudAccumulator) {
	if acc == nil {
		return
	}
	acc.reset()
	fraudAccPool.Put(acc)
	if evt != nil {
		evt.Scratch = nil
	}
}

func fraudAccFromEvent(evt *domain.Event) (*fraudAccumulator, bool) {
	if evt == nil || evt.Scratch == nil {
		return nil, false
	}
	if scratchIsOpenRTB(evt.Scratch) {
		return nil, false
	}
	acc := (*fraudAccumulator)(evt.Scratch)
	if acc.magic != fraudAccumulatorMagic {
		return nil, false
	}
	return acc, true
}

func addFraudSignal(evt *domain.Event, id FraudReasonID) {
	acc, ok := fraudAccFromEvent(evt)
	if !ok {
		return
	}
	acc.add(id)
}

func MapFraudTier(score uint8, pass, suspect, ivt, block uint8) FraudTier {
	if pass == 0 && suspect == 0 && ivt == 0 {
		pass = domain.DefaultFraudThresholdPass
		suspect = domain.DefaultFraudThresholdSuspect
		ivt = domain.DefaultFraudThresholdIVT
	}
	if score <= pass {
		return FraudTierPass
	}
	if score <= suspect {
		return FraudTierSuspect
	}
	if score <= ivt {
		return FraudTierIVT
	}
	_ = block
	return FraudTierBlock
}

func fraudThresholdsFromCampaign(camp *domain.Campaign) (pass, suspect, ivt, block uint8) {
	if camp == nil {
		return domain.DefaultFraudThresholdPass, domain.DefaultFraudThresholdSuspect,
			domain.DefaultFraudThresholdIVT, domain.DefaultFraudThresholdBlock
	}
	pass = camp.FraudThresholdPass
	suspect = camp.FraudThresholdSuspect
	ivt = camp.FraudThresholdIVT
	block = camp.FraudThresholdBlock
	if pass == 0 && suspect == 0 && ivt == 0 && block == 0 {
		return domain.DefaultFraudThresholdPass, domain.DefaultFraudThresholdSuspect,
			domain.DefaultFraudThresholdIVT, domain.DefaultFraudThresholdBlock
	}
	return pass, suspect, ivt, block
}

func applyFraudAccumulatorForCampaign(evt *domain.Event, acc *fraudAccumulator, camp *domain.Campaign) FraudTier {
	if evt == nil || acc == nil || acc.count == 0 {
		if evt != nil {
			evt.FraudScore = 0
			evt.FraudReason = ""
			evt.LayerDesyncCount = 0
		}
		return FraudTierPass
	}

	evt.FraudScore = acc.score
	evt.LayerDesyncCount = acc.layerDesyncCount()

	totalLen := 0
	for i := uint8(0); i < acc.count; i++ {
		if i > 0 {
			totalLen++
		}
		totalLen += len(FraudReasonCode(acc.signals[i]))
	}
	if cap(evt.StringBuffer) < totalLen {
		evt.StringBuffer = make([]byte, 0, totalLen+16)
	} else {
		evt.StringBuffer = evt.StringBuffer[:0]
	}
	for i := uint8(0); i < acc.count; i++ {
		if i > 0 {
			evt.StringBuffer = append(evt.StringBuffer, ',')
		}
		evt.StringBuffer = append(evt.StringBuffer, FraudReasonCode(acc.signals[i])...)
	}
	evt.FraudReason = UnsafeString(evt.StringBuffer)

	pass, suspect, ivt, block := fraudThresholdsFromCampaign(camp)
	return MapFraudTier(uint8(acc.score), pass, suspect, ivt, block)
}

func filterDeadlineMonoEvt(evt *domain.Event, ctx context.Context) (int64, bool) {
	if evt != nil && evt.FilterDeadlineMono > 0 {
		return evt.FilterDeadlineMono, true
	}
	return filterDeadlineMonoFromContext(ctx)
}

func filterDeadlineExceededEvt(evt *domain.Event, ctx context.Context) bool {
	if d, ok := filterDeadlineMonoEvt(evt, ctx); ok {
		return MonotonicNano() > d
	}
	return false
}

func filterDeadlineRemainingEvt(evt *domain.Event, ctx context.Context) (time.Duration, bool) {
	d, ok := filterDeadlineMonoEvt(evt, ctx)
	if !ok {
		return 0, false
	}
	rem := d - MonotonicNano()
	if rem <= 0 {
		return 0, true
	}
	return time.Duration(rem), true
}

func filterDeadlineMonoFromContext(ctx context.Context) (int64, bool) {
	if ctx == nil {
		return 0, false
	}
	d, ok := ctx.Value(filterDeadlineKey{}).(int64)
	return d, ok
}

func filterDeadlineExceeded(ctx context.Context) bool {
	if d, ok := filterDeadlineMonoFromContext(ctx); ok {
		return MonotonicNano() > d
	}
	return false
}

const entitlementsTimezoneCacheShards = 16

type entitlementsTimezoneCache struct {
	shards [entitlementsTimezoneCacheShards]struct {
		mu sync.RWMutex
		m  map[string]*time.Location
	}
}

func (c *entitlementsTimezoneCache) location(timezone string) *time.Location {
	if timezone == "" {
		timezone = "UTC"
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(timezone))
	shard := &c.shards[h.Sum32()%entitlementsTimezoneCacheShards]
	shard.mu.RLock()
	loc, ok := shard.m[timezone]
	shard.mu.RUnlock()
	if ok {
		return loc
	}
	loaded, err := time.LoadLocation(timezone)
	if err != nil {
		loaded = time.UTC
	}
	shard.mu.Lock()
	if shard.m == nil {
		shard.m = make(map[string]*time.Location, 4)
	}
	if cached, exists := shard.m[timezone]; exists {
		shard.mu.Unlock()
		return cached
	}
	shard.m[timezone] = loaded
	shard.mu.Unlock()
	return loaded
}

type FraudAccumulator = fraudAccumulator
