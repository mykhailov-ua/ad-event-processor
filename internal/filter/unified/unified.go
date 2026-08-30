package unified

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/database"
	"ad-event-processor/internal/domain"
	filt "ad-event-processor/internal/filter"
	"ad-event-processor/internal/licensing"
	"ad-event-processor/internal/metrics"
	"ad-event-processor/internal/rtb"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"
)

type UnifiedFilter struct {
	redisShards              []redis.UniversalClient
	sharder                  filt.Sharder
	script                   *redis.Script
	scriptHash               string
	scriptHashAny            any
	registry                 domain.CampaignRegistry
	repo                     domain.CampaignRepository
	geo                      filt.GeoProvider
	geoFloors                atomic.Pointer[map[string]int64]
	rateLimit                int
	rateLimitWindow          time.Duration
	dupTTL                   time.Duration
	idempotencyTTL           time.Duration
	clickAmountMicro         int64
	impressionAmountMicro    int64
	streamName               string
	streamKeyVal             filt.StringVal
	maxStreamLen             int
	rateLimitWindowAny       any
	rateLimitAny             any
	dupTTLAny                any
	idempotencyTTLAny        any
	maxStreamLenAny          any
	clickAmountMicroAny      any
	impressionAmountMicroAny any

	dbHealth               filt.DBHealthChecker
	slaPenaltyActive       atomic.Bool
	p95ThresholdMs         float64
	recoveryEmaMs          float64
	recoveryStableDuration time.Duration
	emaAlpha               float64
	latencySamples         []float64
	latencyIdx             int
	latencyMu              sync.Mutex
	recoveryStartTime      time.Time
	currentEma             float64

	clickAmountMicroHalfAny      any
	impressionAmountMicroHalfAny any
	ttcMinMsAny                  any
	impTSTTLAny                  any
	ttcFailClosedAny             any
	skipBudgetDebitAny           any
	quotaEnabledAny              any
	quotaChunkSizeAny            any
	quotaRefillThresholdPctAny   any
	quotaMode                    string
	localQuotaMode               string
	localQuotaCache              *filt.LocalQuotaCache
	localQuantaLedger            QuantaLedger
	localQuantaStrict            QuantaStrictGate
	localQuantaRefill            QuantaRefillSignaler
	localQuantaPublisher         QuantaDeltaPublisher
	localQuantaStream            QuantaStreamPublisher
	localClickIdem               QuantaClickIdem
	localTTC                     *LocalTTCCache
	roughPacing                  *RoughPacingGate
	settingsWatcher              *filt.SettingsWatcher
	dbLookupTimeout              time.Duration
	postgresFallbackAllowed      bool
	luaMetricsSeq                atomic.Uint64
	fastScript                   *redis.Script
	fastScriptHashAny            any
	rollbackScript               *redis.Script
	rollbackScriptHash           string
	fastPathEnabled              atomic.Bool

	luaDurationObservers        []prometheus.Observer
	luaFastDurationObservers    []prometheus.Observer
	luaFastPathCounters         []prometheus.Counter
	luaFullPathCounters         []prometheus.Counter
	luaNoScriptCounters         []prometheus.Counter
	redisObservability          filt.RedisShardObservability
	regionCode                  uint8
	evalPinWorkers              int
	evalPins                    *filterEvalPin
	breakers                    []*database.RedisBreaker
	filterSlowNs                int64
	evalFallbackGate            chan struct{}
	placementBL                 *filt.PlacementBlacklistFilter
	fraudBL                     *filt.FraudBlacklistFilter
	ingressRPDHandledExternally bool
	cgnatGlobalBypass           bool
	mobileCarrierASN            *filt.MobileCarrierASNTable
	asnLookup                   filt.ASNLookup
}

func (f *UnifiedFilter) SetPlacementBlacklistFilter(p *filt.PlacementBlacklistFilter) {
	if f != nil {
		f.placementBL = p
	}
}

func (f *UnifiedFilter) SetFilterSlowMs(ms int) {
	if ms <= 0 {
		f.filterSlowNs = 0
		return
	}
	f.filterSlowNs = int64(ms) * int64(time.Millisecond)
}

func (f *UnifiedFilter) SetPGFallbackAllowed(allowed bool) {
	f.postgresFallbackAllowed = allowed
}

func (f *UnifiedFilter) SetSettingsWatcher(sw *filt.SettingsWatcher) {
	if f != nil {
		f.settingsWatcher = sw
	}
}

func (f *UnifiedFilter) SetTTCMin(d time.Duration) {
	f.ttcMinMsAny = d.Milliseconds()
	f.impTSTTLAny = int((10 * time.Minute).Seconds())
}

func (f *UnifiedFilter) SetTTCFailClosed(v bool) {
	if v {
		f.ttcFailClosedAny = oneAny
	} else {
		f.ttcFailClosedAny = zeroAny
	}
}

func (f *UnifiedFilter) SetSkipBudgetDebit(skip bool) {
	if skip {
		f.skipBudgetDebitAny = oneAny
	} else {
		f.skipBudgetDebitAny = zeroAny
	}
}

func (f *UnifiedFilter) SkipBudgetDebitAny() any {
	if f == nil {
		return zeroAny
	}
	return f.skipBudgetDebitAny
}

func (f *UnifiedFilter) ClickAmountMicro() int64 {
	if f == nil {
		return 0
	}
	return f.clickAmountMicro
}

func (f *UnifiedFilter) ImpressionAmountMicro() int64 {
	if f == nil {
		return 0
	}
	return f.impressionAmountMicro
}

func (f *UnifiedFilter) LocalQuantaFullSkipEligible(evt *domain.Event, campInfo *domain.Campaign) bool {
	return f.localQuantaFullSkipEligible(evt, campInfo)
}

func (f *UnifiedFilter) SetGeoProvider(geo filt.GeoProvider) {
	f.geo = geo
}

func (f *UnifiedFilter) SetGeoBidFloor(country string, floor int64) {
	old := f.geoFloors.Load()
	capHint := 1
	if old != nil {
		capHint = len(*old) + 1
	}
	next := make(map[string]int64, capHint)
	if old != nil {
		for k, v := range *old {
			next[k] = v
		}
	}
	next[country] = floor
	f.geoFloors.Store(&next)
}

func ParseBidMicro(payload []byte) int64 {
	n := len(payload)
	if n < 11 {
		return 0
	}
	_ = payload[n-1]

	for i := 0; i <= n-11; i++ {
		if payload[i] != '"' || loadU64(payload[i:]) != 0x63696d5f64696222 ||
			payload[i+8] != 'r' || payload[i+9] != 'o' {
			continue
		}
		idx := i + 10
		if idx >= n || payload[idx] != '"' {
			continue
		}
		idx++
		for idx < n && (payload[idx] == ' ' || payload[idx] == '\t' || payload[idx] == ':') {
			if payload[idx] == ':' {
				idx++
				break
			}
			idx++
		}

		for idx < n && (payload[idx] == ' ' || payload[idx] == '\t') {
			idx++
		}

		var val int64
		hasDigit := false
		for idx < n && payload[idx] >= '0' && payload[idx] <= '9' {
			val = val*10 + int64(payload[idx]-'0')
			idx++
			hasDigit = true
		}
		if hasDigit {
			return val
		}
		return 0
	}
	return 0
}

func NewUnifiedFilter(
	redisShards []redis.UniversalClient,
	sharder filt.Sharder,
	registry domain.CampaignRegistry,
	repo domain.CampaignRepository,
	rateLimit int,
	rateLimitWindow time.Duration,
	dupTTL time.Duration,
	idempotencyTTL time.Duration,
	clickAmount int64,
	impressionAmount int64,
	streamName string,
	maxStreamLen int,
) *UnifiedFilter {
	_ = InitUnifiedFilterLua()
	script := redis.NewScript(unifiedFilterLuaForScript())
	fastScript := redis.NewScript(budgetFastLua)
	rollbackScript := redis.NewScript(budgetRollbackLua)
	emptyGeoFloors := make(map[string]int64)
	f := &UnifiedFilter{
		redisShards:                  redisShards,
		sharder:                      sharder,
		script:                       script,
		scriptHash:                   script.Hash(),
		scriptHashAny:                script.Hash(),
		fastScript:                   fastScript,
		fastScriptHashAny:            fastScript.Hash(),
		rollbackScript:               rollbackScript,
		rollbackScriptHash:           rollbackScript.Hash(),
		registry:                     registry,
		repo:                         repo,
		rateLimit:                    rateLimit,
		rateLimitWindow:              rateLimitWindow,
		dupTTL:                       dupTTL,
		idempotencyTTL:               idempotencyTTL,
		clickAmountMicro:             clickAmount,
		impressionAmountMicro:        impressionAmount,
		streamName:                   streamName,
		streamKeyVal:                 filt.StringVal{S: streamName},
		maxStreamLen:                 maxStreamLen,
		rateLimitWindowAny:           int(rateLimitWindow.Seconds()),
		rateLimitAny:                 rateLimit,
		dupTTLAny:                    int(dupTTL.Seconds()),
		idempotencyTTLAny:            int(idempotencyTTL.Seconds()),
		maxStreamLenAny:              maxStreamLen,
		clickAmountMicroAny:          clickAmount,
		impressionAmountMicroAny:     impressionAmount,
		clickAmountMicroHalfAny:      clickAmount / 2,
		impressionAmountMicroHalfAny: impressionAmount / 2,
		ttcFailClosedAny:             zeroAny,
		skipBudgetDebitAny:           zeroAny,
		quotaEnabledAny:              zeroAny,
		quotaChunkSizeAny:            zeroAny,
		quotaRefillThresholdPctAny:   20,
		quotaMode:                    "off",
		localQuotaCache:              filt.NewLocalQuotaCache(),
		luaDurationObservers:         newRedisLuaObservers(len(redisShards)),
		luaFastDurationObservers:     newRedisLuaTierObservers(len(redisShards)),
		luaFastPathCounters:          newRedisLuaPathCounters(len(redisShards), true),
		luaFullPathCounters:          newRedisLuaPathCounters(len(redisShards), false),
		luaNoScriptCounters:          newRedisLuaNoScriptCounters(len(redisShards)),
		redisObservability:           filt.NewRedisShardObservability(len(redisShards), filt.LuaMetricsSampleMask),
		dbLookupTimeout:              2 * time.Second,
		postgresFallbackAllowed:      true,
		evalFallbackGate:             make(chan struct{}, 32),
	}
	f.geoFloors.Store(&emptyGeoFloors)
	return f
}

func (f *UnifiedFilter) StreamDeferredToProducer() bool {
	if f == nil {
		return false
	}
	return f.streamKeyVal.S == fcapIgnoredKeyVal.S
}

// SetDeferStreamToProducer switches Lua KEYS[9] to fcap:ignored so unified-filter.lua
// skips XADD; StreamProducer or BrokerProducer in ingest is the sole stream writer.
// localQuantaStream must use the same sentinel to avoid dual XADD on full-skip traffic.
func (f *UnifiedFilter) SetDeferStreamToProducer(deferWrite bool) {
	if f == nil {
		return
	}
	if deferWrite {
		f.streamKeyVal = fcapIgnoredKeyVal
		if f.localQuantaStream != nil {
			f.localQuantaStream.SetStreamName("fcap:ignored")
		}
	} else {
		f.streamKeyVal = filt.StringVal{S: f.streamName}
		if f.localQuantaStream != nil {
			f.localQuantaStream.SetStreamName(f.streamName)
		}
	}
}

func (f *UnifiedFilter) StartScriptPreheater(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = 30 * time.Second
	}
	go func() {
		_ = f.PreloadScripts(ctx)

		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				_ = f.PreloadScripts(ctx)
			}
		}
	}()
}

func (f *UnifiedFilter) SetMetricsSampleMask(mask int) {
	f.redisObservability.SetSampleMask(filt.HistogramSampleMaskFromConfig(mask))
}

func (f *UnifiedFilter) SetRegionCode(code uint8) {
	f.regionCode = code
}

func (f *UnifiedFilter) SetLuaFastPathEnabled(v bool) {
	f.fastPathEnabled.Store(v)
}

func (f *UnifiedFilter) SetQuotaConfig(mode string, chunkSize int64, thresholdPct int) {
	f.quotaMode = mode
	switch mode {
	case "shadow", "live":
		f.quotaEnabledAny = oneAny
	default:
		f.quotaEnabledAny = zeroAny
	}
	f.quotaChunkSizeAny = chunkSize
	if thresholdPct <= 0 {
		thresholdPct = 20
	}
	f.quotaRefillThresholdPctAny = thresholdPct
}

func (f *UnifiedFilter) SetSLATargets(p95, recovery float64, stable time.Duration, alpha float64) {
	f.p95ThresholdMs = p95
	f.recoveryEmaMs = recovery
	f.recoveryStableDuration = stable
	f.emaAlpha = alpha
}

func (f *UnifiedFilter) SLAPenaltyActive() *atomic.Bool {
	return &f.slaPenaltyActive
}

func (f *UnifiedFilter) ResizeTrackers(size int) {
	f.latencyMu.Lock()
	defer f.latencyMu.Unlock()
	f.latencySamples = make([]float64, size)
	f.latencyIdx = 0
}

func (f *UnifiedFilter) SetDBHealthChecker(checker filt.DBHealthChecker) {
	f.dbHealth = checker
}

func (f *UnifiedFilter) StartSLASentinel(ctx context.Context, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if f.dbHealth == nil {
					continue
				}

				start := time.Now()
				pingCtx, pingCancel := context.WithTimeout(ctx, interval)
				err := f.dbHealth.Ping(pingCtx)
				pingCancel()
				latency := float64(time.Since(start).Milliseconds())
				if err != nil {
					latency = f.p95ThresholdMs + 1000
				}

				f.latencyMu.Lock()
				if len(f.latencySamples) > 0 {
					f.latencySamples[f.latencyIdx%len(f.latencySamples)] = latency
					f.latencyIdx++
				}

				if f.currentEma == 0 {
					f.currentEma = latency
				} else {
					f.currentEma = f.emaAlpha*latency + (1-f.emaAlpha)*f.currentEma
				}

				var p95 float64
				if len(f.latencySamples) > 0 {
					samples := make([]float64, len(f.latencySamples))
					copy(samples, f.latencySamples)
					sort.Float64s(samples)
					idx := int(float64(len(samples)) * 0.95)
					if idx >= len(samples) {
						idx = len(samples) - 1
					}
					p95 = samples[idx]
				}

				isActive := f.slaPenaltyActive.Load()

				if !isActive && p95 > f.p95ThresholdMs {

					for _, redisClient := range f.redisShards {
						_ = redisClient.Set(ctx, "sla:penalty:active", true, 0).Err()
					}
					f.slaPenaltyActive.Store(true)
				} else if isActive {
					if f.currentEma < f.recoveryEmaMs {
						if f.recoveryStartTime.IsZero() {
							f.recoveryStartTime = time.Now()
						} else if time.Since(f.recoveryStartTime) >= f.recoveryStableDuration {
							for _, redisClient := range f.redisShards {
								_ = redisClient.Del(ctx, "sla:penalty:active").Err()
							}
							f.slaPenaltyActive.Store(false)
							f.recoveryStartTime = time.Time{}
						}
					} else {
						f.recoveryStartTime = time.Time{}
					}
				}
				f.latencyMu.Unlock()
			}
		}
	}()
}

func (f *UnifiedFilter) checkGeoBidFloor(evt *domain.Event) error {
	country := evt.GeoCountry
	if country == "" {
		if evt.IngestGeoResolved {
			return nil
		}
		var err error
		country, err = f.geo.GetCountry(evt.IP)
		if err != nil || country == "" {
			return nil
		}
	}
	floorPtr := f.geoFloors.Load()
	if floorPtr == nil {
		return nil
	}
	floor, ok := (*floorPtr)[country]
	if !ok {
		return nil
	}
	if floor <= 0 {
		return nil
	}
	if ParseBidMicro(evt.Payload) < floor {
		return filt.ErrBidFloorNotMet
	}
	return nil
}

func (f *UnifiedFilter) CheckFreqLimitGo(evt *domain.Event, campInfo *domain.Campaign) (bool, error) {
	return f.checkFreqLimitGo(evt, campInfo)
}

func (f *UnifiedFilter) checkFreqLimitGo(evt *domain.Event, campInfo *domain.Campaign) (bool, error) {
	if campInfo == nil || campInfo.FreqLimit <= 0 || evt.UserID == "" {
		return false, nil
	}
	if f.settingsWatcher == nil {
		return false, nil
	}
	snap := f.settingsWatcher.GetFcapRtbSnapshot()
	if snap == nil {
		return false, nil
	}
	prefixHash := rtb.HashBytes64([]byte(campInfo.FcapKeyPrefix))
	userHash := rtb.HashBytes64([]byte(evt.UserID))
	count, ok := snap.FcapCount(prefixHash, userHash)
	if ok && count >= uint32(campInfo.FreqLimit) {
		return true, filt.ErrFreqLimitExceeded
	}
	return false, nil
}

func (f *UnifiedFilter) SetRegistry(reg domain.CampaignRegistry) {
	if f != nil {
		f.registry = reg
	}
}

func (f *UnifiedFilter) getCampaign(evt *domain.Event) (*domain.Campaign, bool) {
	if f == nil || f.registry == nil {
		return nil, false
	}
	return filt.GetCampaignFromEvent(f.registry, evt)
}

func (f *UnifiedFilter) Check(ctx context.Context, evt *domain.Event) error {
	if evt != nil && evt.SmokeEvent {
		prevSkip := f.skipBudgetDebitAny
		prevStream := f.streamKeyVal
		f.skipBudgetDebitAny = oneAny
		f.streamKeyVal = fcapIgnoredKeyVal
		err := f.checkPass(ctx, evt)
		f.skipBudgetDebitAny = prevSkip
		f.streamKeyVal = prevStream
		return err
	}
	return f.checkPass(ctx, evt)
}

// checkPass runs after ingest tryAcquireStreamAdmission (TryReserve on stream/broker).
// Debit paths here assume producer capacity is reserved; post-debit enqueue failure
// in ingest must call RollbackDebit to undo Redis or local-quanta spend.
func (f *UnifiedFilter) checkPass(ctx context.Context, evt *domain.Event) error {
	nowNano := filt.MonotonicNano()
	if f.quotaMode == "live" && f.localQuotaCache.IsBlocked(evt.CampaignID, nowNano) {
		metrics.TrackerLocalQuotaBlockTotal.Inc()
		return filt.ErrBudgetExhausted
	}

	campInfo, ok := f.getCampaign(evt)
	if !ok {
		if reg, ok := f.registry.(*filt.Registry); ok && reg.IsStaleMode() {
			return filt.ErrRegistryStale
		}
		return filt.ErrCampaignNotFound
	}

	if evt.ClickID == "" {
		id := filt.NewFastUUID()
		filt.AppendUUID(evt.ClickIDBuf[:0], id)
		evt.ClickID = filt.UnsafeString(evt.ClickIDBuf[:])
	}

	f.applyGoTTC(evt)

	if f.placementBL != nil {
		if err := f.placementBL.Check(ctx, evt); err != nil {
			return err
		}
	}

	if f.fraudBL != nil {
		_ = f.fraudBL.Check(ctx, evt)
		if filt.EventHasFraudL3(evt) {
			return nil
		}
	}

	if f.geo != nil {
		if err := f.checkGeoBidFloor(evt); err != nil {
			return err
		}
	}

	amount := f.clickAmountMicroAny
	if evt.Type == "impression" {
		amount = f.impressionAmountMicroAny
	}

	if f.slaPenaltyActive.Load() {
		if evt.Type == "impression" {
			amount = f.impressionAmountMicroHalfAny
		} else {
			amount = f.clickAmountMicroHalfAny
		}
	}

	amountMicro := f.impressionAmountMicro
	if evt.Type != "impression" {
		amountMicro = f.clickAmountMicro
	}
	if f.slaPenaltyActive.Load() {
		amountMicro /= 2
	}

	if err := f.checkGoRoughPacing(evt, campInfo, amountMicro); err != nil {
		return err
	}

	// Local-quanta live full-skip: Go TrySpendDebit plus async stream enqueue, zero sync EVALSHA.
	// Partial local-quanta live still runs budget-fast.lua with skipBudgetDebit for dedup/sync only.
	if handled, err := f.checkLocalQuanta(ctx, evt, campInfo, amountMicro); handled {
		return err
	}

	shard, _, err := f.resolveDebitShard(evt.CampaignID, evt.UserID, evt.ClickID, campInfo)
	if err != nil {
		return err
	}
	redisClient := f.redisShards[shard%len(f.redisShards)]

	var now time.Time
	if campInfo.Location == nil || campInfo.Location == time.UTC {
		now = filt.CachedTimeUTC()
	} else {
		now = filt.CachedTimeIn(campInfo.Location)
	}
	if err := f.applyLuaGoPrechecks(ctx, evt, campInfo, redisClient, now); err != nil {
		return err
	}

	if f.fastPathEnabled.Load() && !f.needsFullLuaPath(evt, campInfo) {
		if campInfo.FreqLimit > 0 && evt.UserID != "" {
			exceeded, err := f.checkFreqLimitGo(evt, campInfo)
			if err != nil {
				return err
			}
			if exceeded {
				return filt.ErrFreqLimitExceeded
			}
		}
		fastScratch := budgetFastScratchPool.Get().(*budgetFastScratch)
		err := f.runBudgetFastLua(ctx, evt, campInfo, amount, redisClient, shard, fastScratch)
		budgetFastScratchPool.Put(fastScratch)
		if err == nil {
			if campInfo.FreqLimit > 0 && evt.UserID != "" {
				fcapKey := campInfo.FcapKeyPrefix + evt.UserID
				go func(parent context.Context, key string, window int32) {
					fcapCtx, cancel := context.WithTimeout(parent, 20*time.Millisecond)
					defer cancel()
					pipe := redisClient.Pipeline()
					pipe.Incr(fcapCtx, key)
					pipe.Expire(fcapCtx, key, time.Duration(window)*time.Second)
					_, _ = pipe.Exec(fcapCtx)
				}(ctx, fcapKey, campInfo.FreqWindow)
			}
		}
		return err
	}

	scratch := UnifiedScratchPool.Get().(*UnifiedCheckScratch)
	scratch.Acquire()
	err = f.runUnifiedLua(ctx, evt, campInfo, amount, redisClient, shard, scratch)
	scratch.Release()
	UnifiedScratchPool.Put(scratch)
	return err
}

func (f *UnifiedFilter) runUnifiedLua(
	ctx context.Context,
	evt *domain.Event,
	campInfo *domain.Campaign,
	amount any,
	redisClient redis.UniversalClient,
	shard int,
	scratch *UnifiedCheckScratch,
) error {
	if evt == nil {
		return errors.New("unified filter: nil event")
	}

	wDup := &scratch.wDup
	wIdem := &scratch.wIdem
	wDate := &scratch.wDate
	wDS := &scratch.wDS
	wFcap := &scratch.wFcap
	wImpTS := &scratch.wImpTS
	wQuota := &scratch.wQuota
	wRefillLock := &scratch.wRefillLock
	args := scratch.args
	wrappers := &scratch.wrappers

	wDup.Buf = wDup.Buf[:0]
	wDup.Buf = filt.AppendCampaignHashTag(wDup.Buf, evt.CampaignID)
	wDup.Buf = append(wDup.Buf, "dup:"...)
	wDup.Buf = append(wDup.Buf, evt.Type...)
	wDup.Buf = append(wDup.Buf, ':')
	wDup.Buf = append(wDup.Buf, evt.ClickID...)
	dupKey := filt.UnsafeString(wDup.Buf)

	budgetSourceKey := campInfo.BudgetCampaignKey

	wIdem.Buf = wIdem.Buf[:0]
	wIdem.Buf = filt.AppendCampaignHashTag(wIdem.Buf, evt.CampaignID)
	wIdem.Buf = append(wIdem.Buf, "idempotency:click:"...)
	wIdem.Buf = append(wIdem.Buf, evt.ClickID...)
	idempotencyKey := filt.UnsafeString(wIdem.Buf)

	campaignSyncKey := campInfo.CampaignSyncKey
	customerSyncKey := campInfo.CustomerSyncKey

	var now time.Time
	if campInfo.Location == nil || campInfo.Location == time.UTC {
		now = filt.CachedTimeUTC()
	} else {
		now = filt.CachedTimeIn(campInfo.Location)
	}

	wDate.Buf = wDate.Buf[:0]
	wDate.Buf = filt.AppendDate(wDate.Buf, now)
	currentDate := filt.UnsafeString(wDate.Buf)

	wDS.Buf = wDS.Buf[:0]
	wDS.Buf = append(wDS.Buf, campInfo.DailySpendKeyPrefix...)
	wDS.Buf = append(wDS.Buf, currentDate...)
	dailySpendKey := filt.UnsafeString(wDS.Buf)

	if evt.UserID != "" {
		wFcap.Buf = wFcap.Buf[:0]
		wFcap.Buf = append(wFcap.Buf, fcapKeyPrefixForDebit(campInfo, evt.UserID, evt.ClickID)...)
		wFcap.Buf = append(wFcap.Buf, evt.UserID...)
	}

	wImpTS.Buf = wImpTS.Buf[:0]
	wImpTS.Buf = filt.AppendCampaignHashTag(wImpTS.Buf, evt.CampaignID)
	wImpTS.Buf = append(wImpTS.Buf, "imp_ts:"...)
	wImpTS.Buf = append(wImpTS.Buf, evt.UserID...)
	wImpTS.Buf = append(wImpTS.Buf, ':')
	wImpTS.Buf = filt.AppendUUID(wImpTS.Buf, evt.CampaignID)
	impTSKey := filt.UnsafeString(wImpTS.Buf)

	subSlot := 0
	if campInfo != nil {
		subSlot = debitSubSlot(campInfo, evt.UserID, evt.ClickID)
	}

	wQuota.Buf = appendBudgetQuotaKey(wQuota.Buf[:0], evt.CampaignID, subSlot)
	quotaKey := filt.UnsafeString(wQuota.Buf)

	wRefillLock.Buf = wRefillLock.Buf[:0]
	wRefillLock.Buf = filt.AppendCampaignHashTag(wRefillLock.Buf, evt.CampaignID)
	wRefillLock.Buf = append(wRefillLock.Buf, "budget:refill_lock:"...)
	wRefillLock.Buf = filt.AppendUUID(wRefillLock.Buf, evt.CampaignID)
	refillLockKey := filt.UnsafeString(wRefillLock.Buf)

	wFence := &scratch.wFence
	wFence.Buf = wFence.Buf[:0]
	wFence.Buf = append(wFence.Buf, filt.MigrationFenceKeyPrefix...)
	wFence.Buf = filt.AppendUUID(wFence.Buf, evt.CampaignID)
	migrationFenceKey := filt.UnsafeString(wFence.Buf)

	wFrozen := &scratch.wFrozen
	wFrozen.Buf = wFrozen.Buf[:0]
	wFrozen.Buf = append(wFrozen.Buf, filt.BudgetFrozenKeyPrefix...)
	wFrozen.Buf = filt.AppendUUID(wFrozen.Buf, evt.CampaignID)
	budgetFrozenKey := filt.UnsafeString(wFrozen.Buf)

	kv := scratch.keyVals[:]
	kv[1].S = dupKey
	kv[2].S = budgetSourceKey
	kv[3].S = idempotencyKey
	kv[4].S = campaignSyncKey
	kv[5].S = customerSyncKey
	kv[9].S = dailySpendKey
	kv[11].S = impTSKey
	kv[12].S = quotaKey
	kv[13].S = refillLockKey
	kv[14].S = migrationFenceKey
	kv[15].S = budgetFrozenKey

	keyArgs := scratch.keyArgs
	keyArgs[0] = &fcapIgnoredKeyVal
	keyArgs[1] = &kv[1]
	keyArgs[2] = &kv[2]
	keyArgs[3] = &kv[3]
	keyArgs[4] = &kv[4]
	keyArgs[5] = &kv[5]
	keyArgs[6] = &dirtyCampaignsKeyVal
	keyArgs[7] = &dirtyCustomersKeyVal
	// KEYS[9]: real stream name or fcap:ignored when SetDeferStreamToProducer(true).
	keyArgs[8] = &f.streamKeyVal
	keyArgs[9] = &kv[9]
	keyArgs[11] = &kv[11]
	keyArgs[12] = &kv[12]
	keyArgs[13] = &kv[13]
	keyArgs[14] = &refillNeededKeyVal
	keyArgs[15] = &kv[14]
	keyArgs[16] = &kv[15]
	fillLuaIgnoredPrecheckKeys(keyArgs[:], 17, 18)
	if evt.UserID != "" {
		kv[10].S = filt.UnsafeString(wFcap.Buf)
		keyArgs[10] = &kv[10]
	} else {
		keyArgs[10] = &fcapIgnoredKeyVal
	}

	isEven := zeroAny
	if campInfo.PacingMode == domain.PacingModeEven {
		isEven = oneAny
	}

	hr := now.Hour() + 1
	if hr < 1 {
		hr = 1
	} else if hr > 24 {
		hr = 24
	}
	currentHour := hourAnyCache[hr]

	wrappers.clickID.S = evt.ClickID
	wrappers.evtType.S = evt.Type
	wrappers.payload.S = filt.UnsafeString(evt.Payload)
	wrappers.ip.S = evt.IP
	wrappers.ua.S = evt.UA
	wrappers.userID.S = evt.UserID
	wrappers.placementID.S = evt.PlacementID

	args[0] = zeroAny
	args[1] = zeroAny
	args[2] = f.dupTTLAny
	args[3] = amount
	args[4] = f.idempotencyTTLAny
	args[5] = campInfo.IDStrAny
	args[6] = campInfo.CustomerIDStrAny
	args[7] = f.maxStreamLenAny
	args[8] = &wrappers.clickID
	args[9] = &wrappers.evtType
	args[10] = &wrappers.payload
	args[11] = &wrappers.ip
	args[12] = &wrappers.ua
	args[13] = isEven
	args[14] = campInfo.DailyBudgetMicroAny
	args[15] = currentHour
	args[16] = &wrappers.userID
	args[17] = campInfo.FreqLimitAny
	args[18] = campInfo.FreqWindowAny
	args[19] = f.ttcMinMsAny
	args[20] = filt.CachedUnixMilliAnyLoad()
	args[21] = f.impTSTTLAny
	args[22] = f.ttcFailClosedAny
	args[23] = f.skipBudgetDebitAny
	args[24] = f.quotaEnabledAny
	args[25] = f.quotaChunkSizeAny
	args[26] = f.quotaRefillThresholdPctAny
	args[27] = campInfo.LuaRoutingEpoch()
	// ARGV[28-29]: monotonic filter deadline (ns) and now; Lua returns 20 (tier degraded)
	// when elapsed exceeds ARGV[30] (luaDegradeThresholdNs, 2ms) inside the script.
	if evt.FilterDeadlineMono <= 0 {
		args[28] = zeroAny
		args[29] = zeroAny
	} else {
		wD := &scratch.wDeadlineMono
		wD.Buf = strconv.AppendInt(wD.Buf[:0], evt.FilterDeadlineMono, 10)
		scratch.deadlineMonoStr.S = filt.UnsafeString(wD.Buf)
		args[28] = &scratch.deadlineMonoStr
		wN := &scratch.wNowMono
		wN.Buf = strconv.AppendInt(wN.Buf[:0], filt.MonotonicNano(), 10)
		scratch.nowMonoStr.S = filt.UnsafeString(wN.Buf)
		args[29] = &scratch.nowMonoStr
	}
	args[30] = luaDegradeThresholdAny
	args[31] = &wrappers.placementID
	args[32] = zeroAny
	if f.localTTC != nil {
		args[33] = oneAny
	} else {
		args[33] = zeroAny
	}

	for i := range 2 {
		seq := f.luaMetricsSeq.Add(1)
		sampleLua := filt.ShouldSampleHistogram(seq, f.redisObservability.SampleMask())
		var luaStart int64
		if sampleLua || f.filterSlowNs > 0 {
			luaStart = filt.MonotonicNano()
		}
		f.redisObservability.RecordLuaOp(shard, evt.CampaignID, sampleLua)
		incRedisLuaTier(f.luaFullPathCounters, shard)
		// Full-path debit: one EVALSHA (unified-filter.lua) per accept; NOSCRIPT falls back to EVAL.
		res, err := f.evalScript(ctx, redisClient, shard, evt, keyArgs, args[:34])

		f.noteLuaEvalDuration(shard, evt.CampaignID, "full", luaStart, sampleLua, false)

		if err != nil {
			return err
		}

		if res == -1 {
			retry, recErr := f.recoverBudgetAfterMiss(ctx, evt, redisClient, budgetSourceKey, i)
			if recErr != nil {
				return recErr
			}
			if retry {
				continue
			}
			return filt.ErrBudgetExhausted
		}

		if handled, handleErr := f.handleLuaResult(ctx, evt, campInfo, amount, redisClient, budgetSourceKey, shard, res, sampleLua); handled {
			return handleErr
		}
	}

	return nil
}

func (f *UnifiedFilter) recordAcceptedSpendIfDebited(shard int, campaignID uuid.UUID, amount any, sample bool) {
	if f.skipBudgetDebitAny == oneAny {
		return
	}
	f.redisObservability.RecordAcceptedSpend(shard, campaignID, spendMicroFromAny(amount), sample)
}

func SpendMicroFromAny(amount any) int64 {
	return spendMicroFromAny(amount)
}

func spendMicroFromAny(amount any) int64 {
	v, ok := amount.(int64)
	if !ok {
		return 0
	}
	return v
}

const sealedUnifiedFilterAssetLabel = licensing.AssetLabelUnifiedFilter

var (
	unifiedFilterLuaMu     sync.RWMutex
	unifiedFilterLuaActive string
)

func InitUnifiedFilterLua() error {
	unifiedFilterLuaMu.RLock()
	if unifiedFilterLuaActive != "" {
		unifiedFilterLuaMu.RUnlock()
		return nil
	}
	unifiedFilterLuaMu.RUnlock()

	src, err := resolveUnifiedFilterLuaSource()
	if err != nil {
		return err
	}
	activateUnifiedFilterLuaSource(src)
	return nil
}

func sealedUnifiedFilterBlobPath() string {
	if v := os.Getenv("AD_EVENT_PROCESSOR_UNIFIED_FILTER_SEALED_BLOB"); v != "" {
		return v
	}
	return filepath.Join("internal", "ingestion", "unified_filter_sealed.bin")
}

func sealedUnifiedFilterBlob() ([]byte, error) {
	path := sealedUnifiedFilterBlobPath()
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, os.ErrNotExist
	}
	return data, nil
}

func ResolveUnifiedFilterLuaSource() (string, error) {
	return resolveUnifiedFilterLuaSource()
}

func resolveUnifiedFilterLuaSource() (string, error) {
	if config.LicenseAssetsUnsealed() {
		return unifiedFilterLua, nil
	}
	sealed, err := sealedUnifiedFilterBlob()
	if err != nil {
		if os.IsNotExist(err) {
			return unifiedFilterLua, nil
		}
		return "", err
	}
	mck, err := licensing.DeriveMCKFromLicenseFile(
		config.LicensePathFromEnv(),
		nil,
		licensing.HostFingerprint(),
	)
	if err != nil {
		metrics.LicenseLuaSealFailTotal.Inc()
		return "", fmt.Errorf("sealed lua mck: %w", err)
	}
	plain, err := licensing.OpenAsset(sealedUnifiedFilterAssetLabel, sealed, mck)
	if err != nil {
		metrics.LicenseLuaSealFailTotal.Inc()
		return "", fmt.Errorf("sealed lua open: %w", err)
	}
	if len(plain) == 0 {
		metrics.LicenseLuaSealFailTotal.Inc()
		return "", licensing.ErrSealFormat
	}
	return string(plain), nil
}

func activateUnifiedFilterLuaSource(src string) {
	unifiedFilterLuaMu.Lock()
	unifiedFilterLuaActive = src
	unifiedFilterLuaAny = src
	unifiedFilterLuaMu.Unlock()
}
