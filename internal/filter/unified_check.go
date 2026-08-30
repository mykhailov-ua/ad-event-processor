package filter

import (
	"context"
	_ "embed"
	"errors"
	"hash/crc32"
	"strings"
	"sync/atomic"
	"time"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/metrics"
	"ad-event-processor/pkg/piihash"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type ScheduleFilter struct {
	registry domain.CampaignRegistry
}

func NewScheduleFilter(registry domain.CampaignRegistry) *ScheduleFilter {
	return &ScheduleFilter{registry: registry}
}

func (f *ScheduleFilter) Check(ctx context.Context, evt *domain.Event) error {
	camp, ok := GetCampaignFromEvent(f.registry, evt)
	if !ok {
		if reg, ok := f.registry.(*Registry); ok && reg.IsStaleMode() {
			return ErrRegistryStale
		}
		return ErrCampaignNotFound
	}
	now := CachedTimeUTC()
	if camp.StartAt != nil && now.Before(*camp.StartAt) {
		return ErrScheduleBlocked
	}
	if camp.EndAt != nil && !now.Before(*camp.EndAt) {
		return ErrScheduleBlocked
	}
	if len(camp.DaypartHours) > 0 {
		if camp.Location == nil {
			return ErrScheduleBlocked
		}
		hour := int16(CachedTimeIn(camp.Location).Hour())
		if _, allowed := camp.DaypartHours[hour]; !allowed {
			return ErrScheduleBlocked
		}
	}
	return nil
}

func DaypartSliceToSet(hours []int16) map[int16]struct{} {
	if len(hours) == 0 {
		return nil
	}
	m := make(map[int16]struct{}, len(hours))
	for _, h := range hours {
		m[h] = struct{}{}
	}
	return m
}

type segmentMemberCacheKey struct {
	segmentID uuid.UUID
	userHash  [16]byte
}

type segmentMemberCacheItem struct {
	member bool
	expiry int64
}

const (
	segmentMemberCacheShards             = 128
	segmentMemberCacheTTL                = 5 * time.Second
	segmentMemberCacheMaxEntriesPerShard = 2048
)

type segmentMemberCacheShard struct {
	snap atomic.Pointer[segmentMemberShardSnapshot]
}

type segmentMemberShardSnapshot struct {
	entries map[segmentMemberCacheKey]segmentMemberCacheItem
}

type segmentMemberCache struct {
	shards [segmentMemberCacheShards]segmentMemberCacheShard
}

func newSegmentMemberCache() *segmentMemberCache {
	c := &segmentMemberCache{}
	for i := range segmentMemberCacheShards {
		c.shards[i].snap.Store(&segmentMemberShardSnapshot{
			entries: make(map[segmentMemberCacheKey]segmentMemberCacheItem, 64),
		})
	}
	return c
}

func segmentMemberShardStore(shard *segmentMemberCacheShard, key segmentMemberCacheKey, item segmentMemberCacheItem, nowMs int64) {
	for {
		old := shard.snap.Load()
		next := segmentMemberCloneEntries(old, nowMs, key, item)
		newSnap := &segmentMemberShardSnapshot{entries: next}
		if shard.snap.CompareAndSwap(old, newSnap) {
			return
		}
	}
}

func segmentMemberShardDelete(shard *segmentMemberCacheShard, key segmentMemberCacheKey) {
	for {
		old := shard.snap.Load()
		if old == nil {
			return
		}
		if _, ok := old.entries[key]; !ok {
			return
		}
		next := make(map[segmentMemberCacheKey]segmentMemberCacheItem, len(old.entries)-1)
		for k, v := range old.entries {
			if k != key {
				next[k] = v
			}
		}
		newSnap := &segmentMemberShardSnapshot{entries: next}
		if shard.snap.CompareAndSwap(old, newSnap) {
			return
		}
	}
}

func segmentMemberCloneEntries(old *segmentMemberShardSnapshot, nowMs int64, key segmentMemberCacheKey, item segmentMemberCacheItem) map[segmentMemberCacheKey]segmentMemberCacheItem {
	var oldMap map[segmentMemberCacheKey]segmentMemberCacheItem
	if old != nil {
		oldMap = old.entries
	}
	next := make(map[segmentMemberCacheKey]segmentMemberCacheItem, len(oldMap)+1)
	for k, v := range oldMap {
		if nowMs < v.expiry {
			next[k] = v
		}
	}
	if len(next) >= segmentMemberCacheMaxEntriesPerShard {
		segmentMemberCachePruneMap(next, nowMs)
	}
	next[key] = item
	return next
}

func segmentMemberCachePruneMap(entries map[segmentMemberCacheKey]segmentMemberCacheItem, nowMs int64) {
	for k, v := range entries {
		if nowMs >= v.expiry {
			delete(entries, k)
		}
	}
	for len(entries) >= segmentMemberCacheMaxEntriesPerShard {
		for k := range entries {
			delete(entries, k)
			break
		}
	}
}

func segmentMemberCacheShardIndex(segmentID uuid.UUID, userHash [16]byte) uint32 {
	h := uint32(segmentID[0]) | (uint32(segmentID[1]) << 8)
	h ^= uint32(userHash[0]) | (uint32(userHash[1]) << 8)
	return h % segmentMemberCacheShards
}

func (c *segmentMemberCache) invalidate(segmentID uuid.UUID, userHash [16]byte) {
	if c == nil {
		return
	}
	key := segmentMemberCacheKey{segmentID: segmentID, userHash: userHash}
	shard := &c.shards[segmentMemberCacheShardIndex(segmentID, userHash)]
	segmentMemberShardDelete(shard, key)
}

func (c *segmentMemberCache) memberExists(
	ctx context.Context,
	redisShards []redis.UniversalClient,
	segmentID uuid.UUID,
	userHash [16]byte,
) (bool, error) {
	if c == nil {
		return segmentMemberExists(ctx, redisShards, segmentID, userHash)
	}
	key := segmentMemberCacheKey{segmentID: segmentID, userHash: userHash}
	shardIdx := segmentMemberCacheShardIndex(segmentID, userHash)
	shard := &c.shards[shardIdx]

	nowMs := CachedUnixMilliNow()
	snap := shard.snap.Load()
	if snap != nil {
		if item, ok := snap.entries[key]; ok && nowMs < item.expiry {
			return item.member, nil
		}
	}

	member, err := segmentMemberExists(ctx, redisShards, segmentID, userHash)
	if err != nil {
		return false, err
	}

	segmentMemberShardStore(shard, key, segmentMemberCacheItem{
		member: member,
		expiry: nowMs + segmentMemberCacheTTL.Milliseconds(),
	}, nowMs)
	return member, nil
}

type SegmentFilter struct {
	redisShards []redis.UniversalClient
	registry    domain.CampaignRegistry
	hasher      *piihash.Hasher
	memberCache *segmentMemberCache
}

func NewSegmentFilter(redisShards []redis.UniversalClient, registry domain.CampaignRegistry, hasher *piihash.Hasher) *SegmentFilter {
	return &SegmentFilter{
		redisShards: redisShards,
		registry:    registry,
		hasher:      hasher,
		memberCache: newSegmentMemberCache(),
	}
}

func (f *SegmentFilter) Check(ctx context.Context, evt *domain.Event) error {
	if evt == nil || f.registry == nil {
		return nil
	}
	camp, ok := GetCampaignFromEvent(f.registry, evt)
	if !ok || camp == nil {
		return nil
	}
	if camp.SegmentIncludeID == uuid.Nil && camp.SegmentExcludeID == uuid.Nil {
		return nil
	}
	userHash, ok := SegmentUserHash(f.hasher, evt)
	if !ok {
		if camp.SegmentIncludeID != uuid.Nil {
			return ErrSegmentNotIncluded
		}
		return nil
	}
	if camp.SegmentExcludeID != uuid.Nil {
		member, err := f.memberCache.memberExists(ctx, f.redisShards, camp.SegmentExcludeID, userHash)
		if err != nil {
			return nil
		}
		if member {
			return ErrSegmentExcluded
		}
	}
	if camp.SegmentIncludeID != uuid.Nil {
		member, err := f.memberCache.memberExists(ctx, f.redisShards, camp.SegmentIncludeID, userHash)
		if err != nil {
			return nil
		}
		if !member {
			return ErrSegmentNotIncluded
		}
	}
	return nil
}

func appendHex16(dst []byte, h [16]byte) []byte {
	for i := range 16 {
		dst = append(dst, HexChars[h[i]>>4], HexChars[h[i]&0xf])
	}
	return dst
}

func appendSegmentMemberKey(dst []byte, segmentID uuid.UUID, userHash [16]byte) []byte {
	dst = append(dst, "segment:u:"...)
	dst = AppendUUID(dst, segmentID)
	dst = append(dst, ':')
	return appendHex16(dst, userHash)
}

func pickSegmentShard(redisShards []redis.UniversalClient, segmentID uuid.UUID) redis.UniversalClient {
	if len(redisShards) == 0 {
		return nil
	}
	var h uint32
	for i := range 16 {
		h = h*31 + uint32(segmentID[i])
	}
	start := int(h % uint32(len(redisShards)))
	for i := range redisShards {
		idx := (start + i) % len(redisShards)
		if redisShards[idx] != nil {
			return redisShards[idx]
		}
	}
	return nil
}

func segmentMemberExists(ctx context.Context, redisShards []redis.UniversalClient, segmentID uuid.UUID, userHash [16]byte) (bool, error) {
	redisClient := pickSegmentShard(redisShards, segmentID)
	if redisClient == nil || segmentID == uuid.Nil {
		return false, nil
	}
	w := bufPool.Get().(*bufWrapper)
	w.Buf = appendSegmentMemberKey(w.Buf[:0], segmentID, userHash)
	key := UnsafeString(w.Buf)
	err := redisClient.Get(ctx, key).Err()
	bufPool.Put(w)
	if errors.Is(err, redis.Nil) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func AddSegmentMember(ctx context.Context, redisShards []redis.UniversalClient, segmentID uuid.UUID, userHash [16]byte, ttl time.Duration) error {
	if segmentID == uuid.Nil || ttl <= 0 {
		return nil
	}
	redisClient := pickSegmentShard(redisShards, segmentID)
	if redisClient == nil {
		return nil
	}
	w := bufPool.Get().(*bufWrapper)
	w.Buf = appendSegmentMemberKey(w.Buf[:0], segmentID, userHash)
	key := UnsafeString(w.Buf)
	err := redisClient.Set(ctx, key, "1", ttl).Err()
	bufPool.Put(w)
	return err
}

func SegmentUserHash(hasher *piihash.Hasher, evt *domain.Event) ([16]byte, bool) {
	if evt == nil || evt.UserID == "" {
		return [16]byte{}, false
	}
	if evt.HasUserPIIHash {
		return evt.UserPIIHash, true
	}
	if hasher == nil {
		return [16]byte{}, false
	}
	h := hasher.HashUserID(evt.UserID)
	evt.UserPIIHash = h
	evt.HasUserPIIHash = true
	return h, true
}

const ConversionEventType = "conversion"

type tlsBlocklistSnapshot struct {
	blocked map[uint32]struct{}
}

type DeviceFilter struct {
	settings             *SettingsWatcher
	blockedTLS           atomic.Pointer[tlsBlocklistSnapshot]
	osFingerprintEnabled atomic.Bool
	ja4CorpusEnabled     atomic.Bool
	tcpSynSigEnabled     atomic.Bool
}

func NewDeviceFilter(settings *SettingsWatcher) *DeviceFilter {
	f := &DeviceFilter{settings: settings}
	f.osFingerprintEnabled.Store(true)
	f.reloadBlocklist()
	if settings != nil {
		settings.AddChangeListener(func(_ *DynamicConfig) {
			f.reloadBlocklist()
		})
	}
	return f
}

func (f *DeviceFilter) SetOSFingerprintEnabled(enabled bool) {
	f.osFingerprintEnabled.Store(enabled)
}

func (f *DeviceFilter) SetJA4BrowserCorpusEnabled(enabled bool) {
	f.ja4CorpusEnabled.Store(enabled)
}

func (f *DeviceFilter) SetTCPSynSigEnabled(enabled bool) {
	f.tcpSynSigEnabled.Store(enabled)
}

func (f *DeviceFilter) ReloadBlocklist() {
	f.reloadBlocklist()
}

func (f *DeviceFilter) reloadBlocklist() {
	if f == nil {
		return
	}
	var hashes []string
	if f.settings != nil {
		hashes = parseCommaList(f.settings.Get().TLSHashBlocklist)
	}
	m := make(map[uint32]struct{}, len(hashes))
	for _, h := range hashes {
		m[crc32.ChecksumIEEE([]byte(h))] = struct{}{}
	}
	f.blockedTLS.Store(&tlsBlocklistSnapshot{blocked: m})
}

func (f *DeviceFilter) Check(ctx context.Context, evt *domain.Event) error {
	if evt == nil {
		return nil
	}
	var blocked map[uint32]struct{}
	if snap := f.blockedTLS.Load(); snap != nil {
		blocked = snap.blocked
	}

	if evt.TLSHash != "" && len(blocked) > 0 {
		h := crc32.ChecksumIEEE([]byte(evt.TLSHash))
		if _, onList := blocked[h]; onList {
			AddFraudSignal(evt, FraudReasonTLSBlocklist)
		}
	}
	if deviceHintsMismatch(evt.SecCHUA, evt.UA) {
		AddFraudSignal(evt, FraudReasonDeviceMismatch)
	}
	if TlsFingerprintImpersonating(evt.UA, []byte(evt.TLSJA3), []byte(evt.TLSJA4), []byte(evt.TLSHash)) {
		AddFraudSignal(evt, FraudReasonDeviceMismatch)
	}
	if f.ja4CorpusEnabled.Load() && ja4BrowserCorpusMismatch(evt.UA, []byte(evt.TLSJA4)) {
		AddFraudSignal(evt, FraudReasonTLSJA4Mismatch)
	}
	if f.osFingerprintEnabled.Load() && evt.UA != "" {
		if evt.TCPTTLSet == 0 {
			metrics.OSFingerprintSkippedTotal.WithLabelValues("no_tcp_headers").Inc()
		} else if OsFingerprintMismatch(evt.UA, evt.TCPTTL, evt.TCPWindowSet, evt.TCPWindow) {
			metrics.OSFingerprintMismatchTotal.Inc()
			AddFraudSignal(evt, FraudReasonOSFingerprint)
		}
	}
	if f.tcpSynSigEnabled.Load() && evt.UA != "" {
		if evt.TCPSigSet == 0 {
			metrics.TCPSynSigSkippedTotal.WithLabelValues("no_tcp_sig").Inc()
		} else if TcpSynSigMismatch(evt.UA, evt.TCPSig) {
			metrics.TCPSynSigMismatchTotal.Inc()
			AddFraudSignal(evt, FraudReasonTCPSynOSMismatch)
		}
	}
	return nil
}

func parseCommaList(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

var JA4BrowserCorpusMismatchFn func(ua string, ja4 []byte) bool

func ja4BrowserCorpusMismatch(ua string, ja4 []byte) bool {
	if JA4BrowserCorpusMismatchFn != nil {
		return JA4BrowserCorpusMismatchFn(ua, ja4)
	}
	return false
}

func deviceHintsMismatch(secCHUA, ua string) bool {
	if secCHUA == "" {
		return false
	}
	if ua == "" {
		return true
	}
	if strings.Contains(secCHUA, "Chrome") &&
		!strings.Contains(ua, "Chrome") &&
		!strings.Contains(ua, "Chromium") {
		return true
	}
	return false
}

type BehaviorTelemetryFilter struct {
	registry domain.CampaignRegistry
	enabled  bool
}

func NewBehaviorTelemetryFilter(registry domain.CampaignRegistry) *BehaviorTelemetryFilter {
	return &BehaviorTelemetryFilter{registry: registry}
}

func (f *BehaviorTelemetryFilter) SetEnabled(enabled bool) {
	if f == nil {
		return
	}
	f.enabled = enabled
}

func (f *BehaviorTelemetryFilter) Check(ctx context.Context, evt *domain.Event) error {
	if f == nil || !f.enabled || evt == nil || evt.Type != "conversion" {
		return nil
	}
	if ScanUAFamily(evt.UA) == UAFamilyUnknown {
		return nil
	}
	if f.registry == nil {
		return nil
	}
	camp, ok := f.registry.GetCampaign(evt.CampaignID)
	if !ok || camp == nil || !campaignRequiresBehaviorTelemetry(camp) {
		return nil
	}
	if evt.TelemetrySet == 0 || len(evt.TelemetryEvents) == 0 {
		metrics.BehaviorTelemetryMissingTotal.Inc()
		AddFraudSignal(evt, FraudReasonBehaviorTelemetryMissing)
		return nil
	}
	if CheckBezierBot(behaviorTelemetryToVerifyEvents(evt.TelemetryEvents)) != "" {
		metrics.BehaviorBezierBotTotal.Inc()
		AddFraudSignal(evt, FraudReasonBehaviorBezierBot)
	}
	return nil
}

func campaignRequiresBehaviorTelemetry(camp *domain.Campaign) bool {
	return camp.SafePageEnabled && camp.AttestationEnabled
}

func behaviorTelemetryToVerifyEvents(in []domain.BehaviorTelemetryEvent) []SafePageVerifyEvent {
	if len(in) == 0 {
		return nil
	}
	out := make([]SafePageVerifyEvent, len(in))
	for i := range in {
		out[i] = SafePageVerifyEvent{
			T:  in[i].T,
			TS: in[i].TS,
			X:  in[i].X,
			Y:  in[i].Y,
		}
	}
	return out
}
