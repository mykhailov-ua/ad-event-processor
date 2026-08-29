package filter

import (
	"context"
	_ "embed"
	"log/slog"
	"math/bits"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
	"unsafe"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/licensing"
	"ad-event-processor/internal/metrics"
	"ad-event-processor/pkg/logger"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"
)

type EntitlementsFilter struct {
	registry          *Registry
	sharder           Sharder
	redisShards       []redis.UniversalClient
	regionCode        uint8
	tzCache           entitlementsTimezoneCache
	cgnatGlobalBypass bool
	mobileCarrierASN  *MobileCarrierASNTable
	asnLookup         ASNLookup
}

func NewEntitlementsFilter(registry *Registry, sharder Sharder, redisShards []redis.UniversalClient) *EntitlementsFilter {
	return &EntitlementsFilter{
		registry:    registry,
		sharder:     sharder,
		redisShards: redisShards,
	}
}

func (f *EntitlementsFilter) SetRegionCode(code uint8) {
	if f != nil {
		f.regionCode = code
	}
}

func (f *EntitlementsFilter) ConfigureCGNAT(globalBypass bool, table *MobileCarrierASNTable, lookup ASNLookup) {
	if f == nil {
		return
	}
	f.cgnatGlobalBypass = globalBypass
	f.mobileCarrierASN = table
	f.asnLookup = lookup
}

func (f *EntitlementsFilter) getRedisShardClient(id uuid.UUID) redis.UniversalClient {
	shard := f.sharder.GetShard(id)
	return f.redisShards[shard]
}

func (f *EntitlementsFilter) Check(ctx context.Context, evt *domain.Event) error {
	campInfo, ok := GetCampaignFromEvent(f.registry, evt)
	if !ok {
		return ErrCampaignNotFound
	}
	custID := campInfo.CustomerID

	if evt.Type == "bid" || evt.Type == "rtb" {
		state, depEnt := f.registry.GetLicenseState()
		if !licensing.OpenRTBAllowed(state, depEnt) {
			return ErrLicenseExpired
		}
	}

	ent, ok := f.registry.GetEntitlements(custID)
	if !ok {
		return nil
	}

	if evt.Type == "bid" || evt.Type == "rtb" {
		if !ent.Features.OpenRTBEnabled() {
			return ErrLicenseExpired
		}
	}

	if ent.Limits.MaxRequestsPerDay == 0 {
		return nil
	}

	if CgnatBypassForCampaign(f.cgnatGlobalBypass, f.registry, evt.CampaignID, f.mobileCarrierASN, f.asnLookup, evt.IP, "ingress_rpd") {
		return nil
	}

	timezone := ent.Limits.QuotaResetTimezone
	if timezone == "" {
		timezone = "UTC"
	}

	loc := f.tzCache.location(timezone)

	dateStr := CachedTimeIn(loc).Format("20060102")

	var keyBuf [128]byte
	b := IngressDayKey(keyBuf[:0], f.regionCode, custID, dateStr)
	redisKey := UnsafeString(b)

	redisClient := f.getRedisShardClient(custID)
	if redisClient == nil {
		return nil
	}

	pipe := redisClient.Pipeline()
	incr := pipe.Incr(ctx, redisKey)
	pipe.Expire(ctx, redisKey, 28*time.Hour)
	_, execErr := pipe.Exec(ctx)
	if execErr != nil {
		slog.Warn("failed to increment daily quota counter in Redis", "customer_id", custID, "error", execErr)
		return nil
	}

	currentVal := incr.Val()
	if uint64(currentVal) > ent.Limits.MaxRequestsPerDay {
		return ErrDailyQuotaExceeded
	}

	return nil
}

const fraudBlacklistKey = "blacklist:fraud"

type FraudLayer uint8

const (
	FraudLayerNone FraudLayer = iota
	FraudLayerL2Shadow
	FraudLayerL1Reject
)

func decideFraudLayer(acc *fraudAccumulator, tier FraudTier) FraudLayer {
	if acc == nil || acc.count == 0 {
		return FraudLayerNone
	}
	if acc.hasFlags(FraudSignalL3) {
		return FraudLayerL1Reject
	}
	if acc.countFlags(FraudSignalL1High) >= 2 {
		return FraudLayerL1Reject
	}
	if acc.countFlags(FraudSignalL1High) >= 1 ||
		acc.countFlags(FraudSignalL2Weak) >= 1 ||
		tier == FraudTierSuspect ||
		tier == FraudTierIVT ||
		tier == FraudTierBlock {
		return FraudLayerL2Shadow
	}
	return FraudLayerNone
}

func applyFraudLayerDecision(evt *domain.Event, acc *fraudAccumulator, camp *domain.Campaign, boost uint8) (FraudLayer, error) {
	if evt == nil {
		return FraudLayerNone, nil
	}
	evt.ShadowEvent = false

	if acc != nil && boost > 0 && !acc.boostApplied {
		sum := acc.score + uint32(boost)
		if sum > 100 {
			sum = 100
		}
		acc.score = sum
		acc.boostApplied = true
	}

	tier := applyFraudAccumulatorForCampaign(evt, acc, camp)
	if acc == nil || acc.count == 0 {
		return FraudLayerNone, nil
	}

	layer := decideFraudLayer(acc, tier)
	RecordFraudMetrics(acc, tier, layer)

	switch layer {
	case FraudLayerL1Reject:
		return FraudLayerL1Reject, ErrFraudDetected
	case FraudLayerL2Shadow:
		evt.ShadowEvent = true
		return FraudLayerL2Shadow, nil
	default:
		return FraudLayerNone, nil
	}
}

type fraudBlacklistCacheItem struct {
	blacklisted bool
	expiry      int64
}

const fraudBlacklistCacheTTL = 5 * time.Second

const fraudBlacklistCacheShards = 128

const fraudBlacklistCacheMaxEntriesPerShard = 2048

type fraudBlacklistCacheShard struct {
	snap atomic.Pointer[fraudBlacklistShardSnapshot]
}

type fraudBlacklistShardSnapshot struct {
	entries map[string]fraudBlacklistCacheItem
}

type FraudBlacklistFilter struct {
	redisShards []redis.UniversalClient
	shards      [fraudBlacklistCacheShards]fraudBlacklistCacheShard
}

func NewFraudBlacklistFilter(redisShards []redis.UniversalClient) *FraudBlacklistFilter {
	if len(redisShards) == 0 {
		return nil
	}
	f := &FraudBlacklistFilter{redisShards: redisShards}
	for i := range fraudBlacklistCacheShards {
		f.shards[i].snap.Store(&fraudBlacklistShardSnapshot{
			entries: make(map[string]fraudBlacklistCacheItem, 64),
		})
	}
	return f
}

func fraudBlacklistShardIndex(ip string) uint32 {
	if len(ip) == 0 {
		return 0
	}
	h := uint32(ip[0])
	if len(ip) > 1 {
		h |= uint32(ip[1]) << 8
	}
	return h % fraudBlacklistCacheShards
}

func (f *FraudBlacklistFilter) Check(ctx context.Context, evt *domain.Event) error {
	if f == nil || evt == nil || evt.IP == "" {
		return nil
	}

	ip := evt.IP
	shardIdx := fraudBlacklistShardIndex(ip)
	shard := &f.shards[shardIdx]

	nowMs := CachedUnixMilliNow()
	snap := shard.snap.Load()
	if snap != nil {
		if item, ok := snap.entries[ip]; ok && nowMs < item.expiry {
			if item.blacklisted {
				addFraudSignal(evt, FraudReasonL3Blocklist)
			}
			return nil
		}
	}

	redisClient := PickGlobalReadShardForIP(f.redisShards, ip)
	if redisClient == nil {
		return nil
	}

	onList, err := redisClient.SIsMember(ctx, fraudBlacklistKey, ip).Result()
	if err != nil {
		return nil
	}

	fraudBlacklistShardStore(shard, ip, fraudBlacklistCacheItem{
		blacklisted: onList,
		expiry:      nowMs + fraudBlacklistCacheTTL.Milliseconds(),
	}, nowMs)

	if onList {
		addFraudSignal(evt, FraudReasonL3Blocklist)
	}
	return nil
}

func fraudBlacklistShardStore(shard *fraudBlacklistCacheShard, ip string, item fraudBlacklistCacheItem, nowMs int64) {
	for {
		old := shard.snap.Load()
		next := fraudBlacklistCloneEntries(old, nowMs, ip, item)
		newSnap := &fraudBlacklistShardSnapshot{entries: next}
		if shard.snap.CompareAndSwap(old, newSnap) {
			return
		}
	}
}

func fraudBlacklistShardDeleteIP(shard *fraudBlacklistCacheShard, ip string) {
	for {
		old := shard.snap.Load()
		if old == nil {
			return
		}
		if _, ok := old.entries[ip]; !ok {
			return
		}
		next := make(map[string]fraudBlacklistCacheItem, len(old.entries)-1)
		for k, v := range old.entries {
			if k != ip {
				next[k] = v
			}
		}
		newSnap := &fraudBlacklistShardSnapshot{entries: next}
		if shard.snap.CompareAndSwap(old, newSnap) {
			return
		}
	}
}

func fraudBlacklistCloneEntries(old *fraudBlacklistShardSnapshot, nowMs int64, ip string, item fraudBlacklistCacheItem) map[string]fraudBlacklistCacheItem {
	var oldMap map[string]fraudBlacklistCacheItem
	if old != nil {
		oldMap = old.entries
	}
	next := make(map[string]fraudBlacklistCacheItem, len(oldMap)+1)
	for k, v := range oldMap {
		if nowMs < v.expiry {
			next[k] = v
		}
	}
	if len(next) >= fraudBlacklistCacheMaxEntriesPerShard {
		fraudBlacklistCachePruneMap(next, nowMs)
	}
	next[ip] = item
	return next
}

func fraudBlacklistCachePruneMap(entries map[string]fraudBlacklistCacheItem, now int64) {
	for k, v := range entries {
		if now >= v.expiry {
			delete(entries, k)
		}
	}
	for len(entries) >= fraudBlacklistCacheMaxEntriesPerShard {
		for k := range entries {
			delete(entries, k)
			break
		}
	}
}

const (
	licenseRPSBurstPercent   = 10
	licenseRPSBurstWindowSec = 45
)

type LicenseRPSFilter struct {
	registry LicenseStateReader
}

func NewLicenseRPSFilter(registry LicenseStateReader) *LicenseRPSFilter {
	return &LicenseRPSFilter{registry: registry}
}

type deploymentRPSLimiter struct {
	epoch       atomic.Uint64
	count       atomic.Uint64
	burstRemain atomic.Uint64
	burstInit   atomic.Uint32
}

var globalDeploymentRPS deploymentRPSLimiter

func licenseRPSSoftCeil(maxRPS uint64) uint64 {
	if maxRPS == 0 {
		return 0
	}
	extra := maxRPS * licenseRPSBurstPercent / 100
	if extra == 0 {
		extra = 1
	}
	return maxRPS + extra
}

func licenseRPSBurstCap(maxRPS uint64) uint64 {
	if maxRPS == 0 {
		return 0
	}
	return maxRPS * licenseRPSBurstWindowSec * uint64(licenseRPSBurstPercent) / 100
}

func (l *deploymentRPSLimiter) resetForTests() {
	l.epoch.Store(uint64(time.Now().Unix()))
	l.count.Store(0)
	l.burstRemain.Store(0)
	l.burstInit.Store(0)
}

func (l *deploymentRPSLimiter) ensureBurstPool(burstCap uint64) {
	if burstCap == 0 || l.burstInit.Load() != 0 {
		return
	}
	if l.burstInit.CompareAndSwap(0, 1) {
		l.burstRemain.Store(burstCap)
	}
}

func (l *deploymentRPSLimiter) refillBurst(maxRPS, burstCap, lastCount uint64) {
	if burstCap == 0 || lastCount > maxRPS {
		return
	}
	add := maxRPS * licenseRPSBurstPercent / 100
	if add == 0 {
		add = 1
	}
	for {
		cur := l.burstRemain.Load()
		next := cur + add
		if next > burstCap {
			next = burstCap
		}
		if next == cur || l.burstRemain.CompareAndSwap(cur, next) {
			return
		}
	}
}

func (l *deploymentRPSLimiter) consumeBurst() bool {
	for {
		cur := l.burstRemain.Load()
		if cur == 0 {
			return false
		}
		if l.burstRemain.CompareAndSwap(cur, cur-1) {
			return true
		}
	}
}

func (l *deploymentRPSLimiter) allow(maxRPS uint64) bool {
	if maxRPS == 0 {
		return true
	}
	burstCap := licenseRPSBurstCap(maxRPS)
	soft := licenseRPSSoftCeil(maxRPS)
	l.ensureBurstPool(burstCap)

	now := cachedUnixSec()
	prev := l.epoch.Load()
	if prev != now {
		if l.epoch.CompareAndSwap(prev, now) {
			lastCount := l.count.Load()
			l.count.Store(0)
			l.refillBurst(maxRPS, burstCap, lastCount)
		}
	}

	n := l.count.Add(1)
	if n <= maxRPS {
		return true
	}
	if n > soft {
		return false
	}
	return l.consumeBurst()
}

func (f *LicenseRPSFilter) Check(_ context.Context, _ *domain.Event) error {
	if f == nil || f.registry == nil {
		return nil
	}
	_, ent := f.registry.GetLicenseState()
	maxRPS := ent.Limits.MaxRPS
	if maxRPS == 0 {
		return nil
	}
	if !licensing.SeedGateRPS(maxRPS) {
		metrics.LicenseRPSExceededTotal.Inc()
		return ErrRateLimitExceeded
	}
	if !globalDeploymentRPS.allow(maxRPS) {
		metrics.LicenseRPSExceededTotal.Inc()
		return ErrRateLimitExceeded
	}
	return nil
}

var (
	filterGeoLookupErrors           = metrics.FilterInternalErrors.WithLabelValues("geo_lookup")
	BrandCreativeReplicaParseErrors = metrics.FilterInternalErrors.WithLabelValues("brand_creative_replica")
	BrandCreativeLoadTimeout        = metrics.FilterInternalErrors.WithLabelValues("brand_creative_load_timeout")
	filterFraudStreamWriteErrors    = metrics.FilterInternalErrors.WithLabelValues("fraud_stream_write")
	FilterEngineFailures            = metrics.FilterInternalErrors.WithLabelValues("filter_engine")
	filterGeoDuration               = metrics.FilterGeoDuration
	geoMetricsSeq                   atomic.Uint64
)

const sampledCampaignBuckets = 256

var sampledCampaignBucketLabels [sampledCampaignBuckets]string

func init() {
	for i := range sampledCampaignBucketLabels {
		sampledCampaignBucketLabels[i] = strconv.Itoa(i)
	}
}

type redisShardObservability struct {
	opsCounters             []prometheus.Counter
	sampleMask              uint64
	sampledCampaignCounters [][]prometheus.Counter
	sampledSpendCounters    [][]prometheus.Counter
}

func newRedisShardObservability(numShards int, sampleMask uint64) redisShardObservability {
	if numShards <= 0 {
		numShards = 1
	}
	if sampleMask == 0 {
		sampleMask = luaMetricsSampleMask
	}
	o := redisShardObservability{
		opsCounters:             NewRedisOpsCounters(numShards),
		sampleMask:              sampleMask,
		sampledCampaignCounters: make([][]prometheus.Counter, numShards),
		sampledSpendCounters:    make([][]prometheus.Counter, numShards),
	}
	shardLabel := make([]string, numShards)
	for s := range numShards {
		shardLabel[s] = strconv.Itoa(s)
		o.sampledCampaignCounters[s] = make([]prometheus.Counter, sampledCampaignBuckets)
		o.sampledSpendCounters[s] = make([]prometheus.Counter, sampledCampaignBuckets)
		for b := range sampledCampaignBuckets {
			o.sampledCampaignCounters[s][b] = metrics.RedisCampaignOpsSampledTotal.WithLabelValues(shardLabel[s], sampledCampaignBucketLabels[b])
			o.sampledSpendCounters[s][b] = metrics.TrackerCampaignSpendSampledTotal.WithLabelValues(shardLabel[s], sampledCampaignBucketLabels[b])
		}
	}
	return o
}

func (o *redisShardObservability) SampleMask() uint64 {
	return o.sampleMask
}

func (o *redisShardObservability) SetSampleMask(mask uint64) {
	o.sampleMask = mask
}

func (o *redisShardObservability) RecordLuaOp(shard int, campaignID uuid.UUID, sample bool) {
	IncRedisOps(o.opsCounters, shard)
	if sample {
		recordSampledCampaignOp(o, shard, campaignID)
	}
}

func (o *redisShardObservability) RecordAcceptedSpend(shard int, campaignID uuid.UUID, spendMicro int64, sample bool) {
	if !sample || spendMicro <= 0 {
		return
	}
	recordSampledCampaignSpend(o, shard, campaignID, spendMicro)
}

func sampledCampaignBucket(campaignID uuid.UUID) int {
	return int(campaignID[0]) ^ int(campaignID[15])
}

func recordSampledCampaignOp(o *redisShardObservability, shard int, campaignID uuid.UUID) {
	if len(o.sampledCampaignCounters) == 0 {
		return
	}
	if shard < 0 {
		shard = 0
	}
	if shard >= len(o.sampledCampaignCounters) {
		shard %= len(o.sampledCampaignCounters)
	}
	bucket := sampledCampaignBucket(campaignID)
	o.sampledCampaignCounters[shard][bucket].Inc()
}

func recordSampledCampaignSpend(o *redisShardObservability, shard int, campaignID uuid.UUID, spendMicro int64) {
	if len(o.sampledSpendCounters) == 0 {
		return
	}
	if shard < 0 {
		shard = 0
	}
	if shard >= len(o.sampledSpendCounters) {
		shard %= len(o.sampledSpendCounters)
	}
	bucket := sampledCampaignBucket(campaignID)
	o.sampledSpendCounters[shard][bucket].Add(float64(spendMicro))
}

const filterRejectSampleEventType = "filter_reject"

var filterRejectCountrySampleSeq atomic.Uint64

func normalizeRejectCountry(country string) string {
	if len(country) != 2 {
		return ""
	}
	b0, b1 := country[0], country[1]
	if b0 < 'A' || b0 > 'Z' || b1 < 'A' || b1 > 'Z' {
		return ""
	}
	return country
}

func truncateRejectPlacement(placement string) string {
	if len(placement) <= 64 {
		return placement
	}
	return placement[:64]
}

func appendRejectSamplePayload(dst []byte, kind, placement, country string) []byte {
	dst = append(dst, `{"k":"`...)
	dst = append(dst, kind...)
	dst = append(dst, `","p":"`...)
	dst = append(dst, placement...)
	dst = append(dst, `","c":"`...)
	dst = append(dst, country...)
	dst = append(dst, `"}`...)
	return dst
}

const (
	auditLogSampleMaskDefault = 127
)

type filterRejectMetricSpec struct {
	metricLabel string
}

var filterRejectSpecs = [...]filterRejectMetricSpec{
	FilterRejectEmergencyBreaker:   {"emergency_breaker"},
	FilterRejectRateLimit:          {"rate_limit"},
	FilterRejectDuplicate:          {"duplicate"},
	FilterRejectBudget:             {"budget"},
	FilterRejectPacing:             {"pacing"},
	FilterRejectFreq:               {"freq"},
	FilterRejectGeo:                {"geo"},
	FilterRejectSchedule:           {"schedule"},
	FilterRejectCampaignNotFound:   {"campaign_not_found"},
	FilterRejectBidFloor:           {"bid_floor"},
	FilterRejectTimeout:            {"filter_timeout"},
	FilterRejectFraud:              {"fraud"},
	FilterRejectConsent:            {"consent_denied"},
	FilterRejectInfra:              {"infra_unavailable"},
	FilterRejectLicenseExpired:     {"license_expired"},
	FilterRejectDailyQuotaExceeded: {"daily_quota_exceeded"},
	FilterRejectPlacementBlocked:   {"placement_blocked"},
	FilterRejectSegmentExcluded:    {"segment_excluded"},
	FilterRejectSegmentNotIncluded: {"segment_not_included"},
	FilterRejectRegistryStale:      {"registry_stale"},
	FilterRejectShardUnavailable:   {"shard_unavailable"},
	FilterRejectProducerOverload:   {"producer_overload"},
	FilterRejectFraudBlocked:       {"fraud_blocked"},
}

var writeAuditLogFn func(l *logger.Logger, seq *atomic.Uint64, mask uint64, shardID int, evt *domain.Event)

func SetWriteAuditLog(fn func(l *logger.Logger, seq *atomic.Uint64, mask uint64, shardID int, evt *domain.Event)) {
	writeAuditLogFn = fn
}

func writeAuditLog(l *logger.Logger, seq *atomic.Uint64, mask uint64, shardID int, evt *domain.Event) {
	if writeAuditLogFn != nil {
		writeAuditLogFn(l, seq, mask, shardID, evt)
	}
}

func RecordFilterRejectCountrySample(kind FilterRejectKind, evt *domain.Event, seq *atomic.Uint64, sampleMask uint64) {
	if evt == nil {
		return
	}
	country := normalizeRejectCountry(evt.GeoCountry)
	if country == "" {
		return
	}
	counter := seq
	mask := sampleMask
	if counter == nil {
		counter = &filterRejectCountrySampleSeq
		mask = auditLogSampleMaskDefault
	}
	if !ShouldSampleHistogram(counter.Add(1), mask) {
		return
	}
	reason := filterRejectSpecs[kind].metricLabel
	metrics.FilterRejectCountryTotal.WithLabelValues(reason, country).Inc()
}

func writeFilterRejectSample(
	l *logger.Logger,
	seq *atomic.Uint64,
	sampleMask uint64,
	shardID int,
	evt *domain.Event,
	kind FilterRejectKind,
) {
	if l == nil || evt == nil || seq == nil {
		return
	}
	if !ShouldSampleHistogram(seq.Add(1), sampleMask) {
		return
	}
	placement := truncateRejectPlacement(evt.PlacementID)
	country := normalizeRejectCountry(evt.GeoCountry)
	if placement == "" && country == "" {
		return
	}

	sample := domain.EventPool.Get().(*domain.Event)
	sample.Reset()
	sample.ClickID = evt.ClickID
	sample.CampaignID = evt.CampaignID
	sample.Type = filterRejectSampleEventType
	sample.PlacementID = placement
	sample.GeoCountry = country
	sample.Payload = appendRejectSamplePayload(sample.Payload[:0], filterRejectSpecs[kind].metricLabel, placement, country)
	writeAuditLog(l, seq, 0, shardID, sample)
	domain.EventPool.Put(sample)
}

func RecordFilterRejectDimensions(
	l *logger.Logger,
	seq *atomic.Uint64,
	sampleMask uint64,
	shardID int,
	evt *domain.Event,
	kind FilterRejectKind,
) {
	RecordFilterRejectCountrySample(kind, evt, seq, sampleMask)
	writeFilterRejectSample(l, seq, sampleMask, shardID, evt, kind)
}

type StringVal struct {
	S string
}

func (sv *StringVal) MarshalBinary() ([]byte, error) {
	if len(sv.S) == 0 {
		return nil, nil
	}
	return unsafe.Slice(unsafe.StringData(sv.S), len(sv.S)), nil
}

func AppendDate(dst []byte, t time.Time) []byte {
	year, month, day := t.Date()
	return append(dst,
		byte('0'+year/1000),
		byte('0'+(year/100)%10),
		byte('0'+(year/10)%10),
		byte('0'+year%10),
		byte('0'+int(month)/10),
		byte('0'+int(month)%10),
		byte('0'+day/10),
		byte('0'+day%10),
	)
}

var (
	ZeroAny any = 0
	OneAny  any = 1
)

var HourAnyCache [25]any

func init() {
	for i := 0; i <= 24; i++ {
		HourAnyCache[i] = i
	}
}

const (
	fraudDesyncLayerTCPOS       uint8 = 1 << 0
	fraudDesyncLayerTLSJA4      uint8 = 1 << 1
	fraudDesyncLayerClientHints uint8 = 1 << 2
	fraudDesyncLayerSecFetch    uint8 = 1 << 3
	fraudDesyncLayerH2          uint8 = 1 << 4
)

func fraudDesyncLayerBit(id FraudReasonID) uint8 {
	switch id {
	case FraudReasonTCPSynOSMismatch:
		return fraudDesyncLayerTCPOS
	case FraudReasonTLSJA4Mismatch:
		return fraudDesyncLayerTLSJA4
	case FraudReasonClientHintsMismatch:
		return fraudDesyncLayerClientHints
	case FraudReasonSecFetchAnomaly:
		return fraudDesyncLayerSecFetch
	case FraudReasonH2SettingsMismatch, FraudReasonH2PseudoOrder, FraudReasonH2DowngradeArtifact:
		return fraudDesyncLayerH2
	default:
		return 0
	}
}

func (a *fraudAccumulator) layerDesyncCount() uint8 {
	if a == nil || a.count == 0 {
		return 0
	}
	var mask uint8
	for i := uint8(0); i < a.count; i++ {
		mask |= fraudDesyncLayerBit(a.signals[i])
	}
	return uint8(bits.OnesCount8(mask))
}

var fraudStreamLayerDesyncLabels = [6]string{"0", "1", "2", "3", "4", "5"}

func observeFraudStreamLayerDesync(count uint8) {
	if count > 5 {
		count = 5
	}
	metrics.FraudStreamLayerDesyncTotal.WithLabelValues(fraudStreamLayerDesyncLabels[count]).Inc()
}

type preboundFraudMetrics struct {
	tierPass    prometheus.Counter
	tierSuspect prometheus.Counter
	tierIVT     prometheus.Counter
	tierBlock   prometheus.Counter

	reason [FraudReasonCount]prometheus.Counter

	l1Reject prometheus.Counter
}

var boundFraudMetrics = newPreboundFraudMetrics()

func newPreboundFraudMetrics() preboundFraudMetrics {
	pm := preboundFraudMetrics{
		tierPass:    metrics.FraudTierTotal.WithLabelValues("pass"),
		tierSuspect: metrics.FraudTierTotal.WithLabelValues("suspect"),
		tierIVT:     metrics.FraudTierTotal.WithLabelValues("ivt"),
		tierBlock:   metrics.FraudTierTotal.WithLabelValues("block"),
		l1Reject:    metrics.L1RejectTotal,
	}
	for id := FraudReasonID(1); id < fraudReasonCount; id++ {
		code := FraudReasonCode(id)
		if code != "" {
			pm.reason[id] = metrics.FraudReasonTotal.WithLabelValues(code)
		}
	}
	return pm
}

func (pm *preboundFraudMetrics) tierCounter(tier FraudTier) prometheus.Counter {
	switch tier {
	case FraudTierSuspect:
		return pm.tierSuspect
	case FraudTierIVT:
		return pm.tierIVT
	case FraudTierBlock:
		return pm.tierBlock
	default:
		return pm.tierPass
	}
}

func RecordFraudMetrics(acc *fraudAccumulator, tier FraudTier, layer FraudLayer) {
	if acc == nil || acc.count == 0 {
		return
	}
	metrics.FraudScoreHistogram.Observe(float64(acc.score))
	boundFraudMetrics.tierCounter(tier).Inc()
	for i := uint8(0); i < acc.count; i++ {
		id := acc.signals[i]
		if id > FraudReasonNone && id < fraudReasonCount {
			if c := boundFraudMetrics.reason[id]; c != nil {
				c.Inc()
			}
		}
	}
	if layer == FraudLayerL1Reject {
		boundFraudMetrics.l1Reject.Inc()
	}
}

func parseBlacklistUpdatePayload(payload string) (ip, reason string, ok bool) {
	idx := strings.LastIndex(payload, ":")
	if idx <= 0 || idx >= len(payload)-1 {
		return "", "", false
	}
	return payload[:idx], payload[idx+1:], true
}

func (f *FraudBlacklistFilter) InvalidateIP(ip string) {
	if f == nil || ip == "" {
		return
	}
	shard := &f.shards[fraudBlacklistShardIndex(ip)]
	fraudBlacklistShardDeleteIP(shard, ip)
}

func (f *FraudBlacklistFilter) RunInvalidationSubscriber(ctx context.Context, channel string) {
	if f == nil {
		return
	}
	channel = domain.DefaultBlacklistUpdateChannel(channel)
	backoff := time.Second
	for ctx.Err() == nil {
		if !f.consumeInvalidationUpdates(ctx, channel) {
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
		if backoff < 30*time.Second {
			backoff *= 2
		}
	}
}

func (f *FraudBlacklistFilter) consumeInvalidationUpdates(ctx context.Context, channel string) bool {
	redisClient := PickLocalGlobalShard(f.redisShards)
	if redisClient == nil {
		return true
	}

	pubsub := redisClient.Subscribe(ctx, channel)
	defer func() { _ = pubsub.Close() }()

	for msg := range pubsub.Channel() {
		if ctx.Err() != nil {
			return false
		}
		ip, reason, ok := parseBlacklistUpdatePayload(msg.Payload)
		if !ok || reason != "fraud" {
			continue
		}
		f.InvalidateIP(ip)
	}
	return true
}
