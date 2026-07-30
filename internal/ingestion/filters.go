package ingestion

import (
	"context"
	"errors"
	"sync"
	"time"
	"unsafe"

	"espx/internal/campaignmodel"
	"espx/internal/ingestion/traceprobe"

	"github.com/google/uuid"
	redis "github.com/redis/go-redis/v9"
)

var (
	ErrRateLimitExceeded      = errors.New("rate limit exceeded")
	ErrDuplicateEvent         = errors.New("duplicate event detected")
	ErrBudgetExhausted        = errors.New("budget exhausted")
	ErrCampaignNotFound       = errors.New("campaign not found in registry")
	ErrPacingExhausted        = errors.New("pacing exhausted")
	ErrFreqLimitExceeded      = errors.New("frequency limit exceeded")
	ErrGeoBlocked             = errors.New("geo-targeting blocked")
	ErrScheduleBlocked        = errors.New("outside delivery schedule")
	ErrFraudDetected          = errors.New("fraud detected")
	ErrEmergencyBreakerActive = errors.New("service temporarily unavailable (emergency breaker active)")
	ErrBidFloorNotMet         = errors.New("bid floor not met")
	ErrMigrationFenced        = errors.New("campaign debit fenced")
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
	geo GeoProvider
}

func NewFraudFilter(geo GeoProvider) *FraudFilter {
	return &FraudFilter{
		geo: geo,
	}
}

func (f *FraudFilter) Check(ctx context.Context, evt *campaignmodel.Event) error {
	isAnon, err := f.geo.IsAnonymous(evt.IP)
	if err == nil && isAnon {
		addFraudSignal(evt, FraudReasonDatacenterIP)
	}
	return nil
}

type GeoFilter struct {
	geo      GeoProvider
	registry campaignmodel.CampaignRegistry
}

func NewGeoFilter(geo GeoProvider, registry campaignmodel.CampaignRegistry) *GeoFilter {
	return &GeoFilter{
		geo:      geo,
		registry: registry,
	}
}

func (f *GeoFilter) Check(ctx context.Context, evt *campaignmodel.Event) error {
	start := monotonicNano()
	err := f.checkGeo(evt)
	observeHistogramSampled(&geoMetricsSeq, luaMetricsSampleMask, filterGeoDuration, start)
	return err
}

func (f *GeoFilter) checkGeo(evt *campaignmodel.Event) error {
	camp, ok := f.registry.GetCampaign(evt.CampaignID)
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
	manager          campaignmodel.BudgetManager
	registry         campaignmodel.CampaignRegistry
	clickAmount      int64
	impressionAmount int64
}

func NewBudgetFilter(manager campaignmodel.BudgetManager, registry campaignmodel.CampaignRegistry, clickAmount, impressionAmount int64) *BudgetFilter {
	return &BudgetFilter{
		manager:          manager,
		registry:         registry,
		clickAmount:      clickAmount,
		impressionAmount: impressionAmount,
	}
}

func (f *BudgetFilter) Check(ctx context.Context, evt *campaignmodel.Event) error {
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
	Check(ctx context.Context, evt *campaignmodel.Event) error
}

type FilterEngine struct {
	filters  []EventFilter
	timeout  time.Duration
	registry campaignmodel.CampaignRegistry
	watcher  *SettingsWatcher
}

func NewFilterEngine(timeout time.Duration, filters ...EventFilter) *FilterEngine {
	return &FilterEngine{filters: filters, timeout: timeout}
}

func (e *FilterEngine) SetRegistry(registry campaignmodel.CampaignRegistry) {
	e.registry = registry
}

func (e *FilterEngine) SetSettingsWatcher(watcher *SettingsWatcher) {
	e.watcher = watcher
}

func (e *FilterEngine) Check(ctx context.Context, evt *campaignmodel.Event) error {
	slot := uint32(0)
	if evt != nil {
		slot = uint32(CampaignSlotIndex(evt.CampaignID))
	}
	traceprobe.FilterCheckEnter(slot)
	err := e.checkInner(ctx, evt)
	traceprobe.FilterCheckExit(slot)
	return err
}

func (e *FilterEngine) checkInner(ctx context.Context, evt *campaignmodel.Event) error {
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
		if filterDeadlineExceededEvt(evt, ctx) {
			retErr = ErrFilterTimeout
			break
		}
		if _, ok := f.(*UnifiedFilter); ok && acc.shouldShortCircuitFraudBudget() {
			var camp *campaignmodel.Campaign
			if e.registry != nil && evt != nil {
				camp, _ = e.registry.GetCampaign(evt.CampaignID)
			}
			layer, err := applyFraudLayerDecision(evt, acc, camp, boost)
			if err != nil {
				retErr = err
				break
			}
			if layer == FraudLayerL1Reject {
				retErr = ErrFraudDetected
				break
			}
			if layer == FraudLayerL2Shadow {
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
		var camp *campaignmodel.Campaign
		if e.registry != nil && evt != nil {
			camp, _ = e.registry.GetCampaign(evt.CampaignID)
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
	rdb redis.Cmdable
	ttl time.Duration
}

func NewDuplicateEventFilter(rdb redis.Cmdable, ttl time.Duration) *DuplicateEventFilter {
	return &DuplicateEventFilter{
		rdb: rdb,
		ttl: ttl,
	}
}

func (f *DuplicateEventFilter) Check(ctx context.Context, evt *campaignmodel.Event) error {
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

	ok, err := f.rdb.SetNX(ctx, key, "1", f.ttl).Result()
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

func (f *EmergencyBreakerFilter) Check(ctx context.Context, evt *campaignmodel.Event) error {
	if f.watcher != nil && f.watcher.Get().EmergencyBreaker {
		return ErrEmergencyBreakerActive
	}
	return nil
}

type PlacementBlacklistFilter struct {
	rdbs []redis.UniversalClient
}

func NewPlacementBlacklistFilter(rdbs []redis.UniversalClient) *PlacementBlacklistFilter {
	return &PlacementBlacklistFilter{rdbs: rdbs}
}

func (f *PlacementBlacklistFilter) Check(ctx context.Context, evt *campaignmodel.Event) error {
	if evt == nil || evt.PlacementID == "" {
		return nil
	}
	rdb := pickLocalGlobalShard(f.rdbs)
	if rdb == nil {
		return nil
	}

	w := bufPool.Get().(*bufWrapper)
	w.buf = appendCampaignHashTag(w.buf[:0], evt.CampaignID)
	w.buf = append(w.buf, "blacklist:placement:"...)
	w.buf = appendUUID(w.buf, evt.CampaignID)
	key := unsafeString(w.buf)

	isBlacklisted, err := rdb.HExists(ctx, key, evt.PlacementID).Result()
	bufPool.Put(w)
	if err != nil {
		return nil
	}
	if isBlacklisted {
		return ErrPlacementBlocked
	}
	return nil
}

var ErrPlacementBlocked = errors.New("placement blocked")
