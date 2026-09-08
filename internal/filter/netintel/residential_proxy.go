package netintel

import (
	"context"
	"hash/crc32"
	"net/netip"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
	_ "unsafe"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/metrics"

	"github.com/google/uuid"
)

// ResidentialIntelTable wraps CIDRTable for residential proxy feed (Redis/file reload -> Publish).
type ResidentialIntelTable struct {
	table *CIDRTable
}

func NewResidentialIntelTable() *ResidentialIntelTable {
	return &ResidentialIntelTable{table: NewCIDRTable()}
}

func (t *ResidentialIntelTable) Ready() bool {
	return t != nil && t.table != nil && t.table.Ready()
}

func (t *ResidentialIntelTable) MatchIP(ip string) bool {
	if t == nil || t.table == nil {
		return false
	}
	match, _ := t.table.MatchIP(ip)
	return match
}

func (t *ResidentialIntelTable) PublishPrefixes(prefixes []netip.Prefix, gen uint64) {
	if t == nil || t.table == nil || len(prefixes) == 0 {
		return
	}
	var b cidrBuilder
	root4, root6 := int32(cidrNoIndex), int32(cidrNoIndex)
	for _, p := range prefixes {
		if !p.IsValid() {
			continue
		}
		b.addPrefix(p.Masked(), CIDRFeedOther, &root4, &root6)
	}
	if len(b.prefs) == 0 {
		return
	}
	t.table.Publish(b.snapshot(root4, root6, gen))
}

type residentialProxyPolicy struct {
	ProxyMinEvents             float64
	ProxyMaxCTR                float64
	ProxyMinUsers              float64
	ProxyMinUserClickGap       float64
	ProxyMinEventsPerUser      float64
	ProxyMinImpressionPressure float64
	ProxyMinUsersPerUA         float64
	ProxyMinClicks             float64
}

type residentialProxyRow struct {
	Events      int
	Clicks      int
	UniqueUsers int
	UniqueUAs   int
}

func defaultResidentialProxyPolicy() residentialProxyPolicy {
	return residentialProxyPolicy{
		ProxyMinEvents:             80,
		ProxyMaxCTR:                0.05,
		ProxyMinUsers:              20,
		ProxyMinUserClickGap:       5.0,
		ProxyMinEventsPerUser:      5.0,
		ProxyMinImpressionPressure: 12.0,
		ProxyMinUsersPerUA:         2.5,
		ProxyMinClicks:             2,
	}
}

type residentialProxyMetrics struct {
	events             float64
	clicks             float64
	ctr                float64
	uniqueUsers        float64
	uniqueUAs          float64
	eventsPerUser      float64
	impressionPressure float64
	userClickGap       float64
	usersPerUA         float64
}

func residentialProxyMetricsFromRow(row residentialProxyRow) residentialProxyMetrics {
	events := float64(row.Events)
	clicks := float64(row.Clicks)
	users := float64(row.UniqueUsers)
	uas := float64(row.UniqueUAs)
	return residentialProxyMetrics{
		events:             events,
		clicks:             clicks,
		ctr:                safeRatio(clicks, events),
		uniqueUsers:        users,
		uniqueUAs:          uas,
		eventsPerUser:      safeRatio(events, users),
		impressionPressure: safeRatio(events, clicks+1),
		userClickGap:       safeRatio(users, clicks+1),
		usersPerUA:         safeRatio(users, uas),
	}
}

func safeRatio(numerator, denominator float64) float64 {
	if denominator <= 0 {
		return 0
	}
	return numerator / denominator
}

func residentialProxySignal(row residentialProxyRow, cfg residentialProxyPolicy) bool {
	m := residentialProxyMetricsFromRow(row)
	if m.events < cfg.ProxyMinEvents {
		return false
	}
	if m.ctr > cfg.ProxyMaxCTR {
		return false
	}
	if m.uniqueUsers < cfg.ProxyMinUsers {
		return false
	}
	if m.userClickGap < cfg.ProxyMinUserClickGap {
		return false
	}
	if m.eventsPerUser < cfg.ProxyMinEventsPerUser {
		return false
	}
	if m.impressionPressure < cfg.ProxyMinImpressionPressure {
		return false
	}
	if m.usersPerUA < cfg.ProxyMinUsersPerUA {
		return false
	}
	if m.clicks < cfg.ProxyMinClicks {
		return false
	}
	return true
}

const localQuantaCacheLine = 64

const (
	residentialProxySlotCount = 4096
	residentialProxySlotMask  = residentialProxySlotCount - 1
	residentialProxyMaxProbe  = 16
	residentialProxyDistinct  = 32
	defaultProxyWindow        = 5 * time.Minute
)

type residentialProxyCell struct {
	campaignHash atomic.Uint32
	_            uint32
	windowStart  atomic.Int64
	events       atomic.Uint32
	clicks       atomic.Uint32
	userHashes   [residentialProxyDistinct]atomic.Uint32
	uaHashes     [residentialProxyDistinct]atomic.Uint32
	_            [localQuantaCacheLine*5 - (4 + 4 + 8 + 4 + 4 + residentialProxyDistinct*4*2)]byte
}

type ResidentialProxyRing struct {
	cells    [residentialProxySlotCount]residentialProxyCell
	windowNs atomic.Uint64
	policy   atomic.Pointer[residentialProxyPolicy]
}

func NewResidentialProxyRing() *ResidentialProxyRing {
	r := &ResidentialProxyRing{}
	r.windowNs.Store(uint64(defaultProxyWindow.Nanoseconds()))
	p := defaultResidentialProxyPolicy()
	r.policy.Store(&p)
	return r
}

func (r *ResidentialProxyRing) SetPolicy(p residentialProxyPolicy) {
	r.policy.Store(&p)
}

func (r *ResidentialProxyRing) SetWindow(d time.Duration) {
	if d > 0 {
		r.windowNs.Store(uint64(d.Nanoseconds()))
	}
}

func (r *ResidentialProxyRing) policySnapshot() residentialProxyPolicy {
	p := r.policy.Load()
	if p == nil {
		return defaultResidentialProxyPolicy()
	}
	return *p
}

func HashResidentialProxyUser(s string) uint32 {
	return crc32.ChecksumIEEE([]byte(s))
}

func HashResidentialProxyUA(s string) uint32 {
	if s == "" {
		return 0
	}
	n := len(s)
	if n > 128 {
		n = 128
	}
	return crc32.ChecksumIEEE([]byte(s[:n]))
}

func residentialProxySlotHash(campaignHash uint32) uint32 {
	return campaignHash & residentialProxySlotMask
}

func (r *ResidentialProxyRing) Observe(campaignHash uint32, isClick bool, userHash, uaHash uint32, nowMono int64) (row residentialProxyRow, signal bool) {
	if r == nil {
		return residentialProxyRow{}, false
	}
	window := int64(r.windowNs.Load())
	if window <= 0 {
		window = int64(defaultProxyWindow.Nanoseconds())
	}
	start := residentialProxySlotHash(campaignHash)
	for probe := range residentialProxyMaxProbe {
		idx := (start + uint32(probe)) & residentialProxySlotMask
		cell := &r.cells[idx]
		ch := cell.campaignHash.Load()
		if ch == 0 {
			if !cell.campaignHash.CompareAndSwap(0, campaignHash) {
				continue
			}
			cell.windowStart.Store(nowMono)
		} else if ch != campaignHash {
			continue
		}
		ws := cell.windowStart.Load()
		if nowMono-ws > window {
			cell.windowStart.Store(nowMono)
			cell.events.Store(0)
			cell.clicks.Store(0)
			for i := range cell.userHashes {
				cell.userHashes[i].Store(0)
				cell.uaHashes[i].Store(0)
			}
		}
		if isClick {
			cell.clicks.Add(1)
		}
		cell.events.Add(1)
		if userHash != 0 {
			insertDistinctHash(cell.userHashes[:], userHash)
		}
		if uaHash != 0 {
			insertDistinctHash(cell.uaHashes[:], uaHash)
		}
		row = residentialProxyRow{
			Events:      int(cell.events.Load()),
			Clicks:      int(cell.clicks.Load()),
			UniqueUsers: countDistinctHashes(cell.userHashes[:]),
			UniqueUAs:   countDistinctHashes(cell.uaHashes[:]),
		}
		signal = residentialProxySignal(row, r.policySnapshot())
		return row, signal
	}
	return residentialProxyRow{}, false
}

func insertDistinctHash(slots []atomic.Uint32, hash uint32) {
	for i := range slots {
		if slots[i].Load() == hash {
			return
		}
	}
	for i := range slots {
		if slots[i].CompareAndSwap(0, hash) {
			return
		}
	}
}

func countDistinctHashes(slots []atomic.Uint32) int {
	n := 0
	for i := range slots {
		if slots[i].Load() != 0 {
			n++
		}
	}
	return n
}

func (r *ResidentialProxyRing) SeedForTest(campaignID uuid.UUID, row ResidentialProxyRow) {
	if r == nil {
		return
	}
	campaignHash := domain.CRC32Castagnoli(&campaignID)
	idx := residentialProxySlotHash(campaignHash)
	cell := &r.cells[idx]
	cell.campaignHash.Store(campaignHash)
	cell.windowStart.Store(monotonicNano())
	cell.events.Store(uint32(row.Events))
	cell.clicks.Store(uint32(row.Clicks))
	for i := range residentialProxyDistinct {
		cell.userHashes[i].Store(0)
		cell.uaHashes[i].Store(0)
	}
	for i := 0; i < row.UniqueUsers && i < residentialProxyDistinct; i++ {
		cell.userHashes[i].Store(uint32(i + 1))
	}
	for i := 0; i < row.UniqueUAs && i < residentialProxyDistinct; i++ {
		cell.uaHashes[i].Store(uint32(i + 1000))
	}
}

type ResidentialProxyRow = residentialProxyRow

func DefaultResidentialProxyPolicyForTest() residentialProxyPolicy {
	return defaultResidentialProxyPolicy()
}

func ResidentialProxySignalForTest(row ResidentialProxyRow, cfg residentialProxyPolicy) bool {
	return residentialProxySignal(row, cfg)
}

func ResidentialProxyPolicyFromEnv() residentialProxyPolicy {
	return residentialProxyPolicyFromEnv()
}

func residentialProxyPolicyFromEnv() residentialProxyPolicy {
	def := defaultResidentialProxyPolicy()
	return residentialProxyPolicy{
		ProxyMinEvents:             envFloatPolicy("FRAUD_POLICY_PROXY_MIN_EVENTS", def.ProxyMinEvents),
		ProxyMaxCTR:                envFloatPolicy("FRAUD_POLICY_PROXY_MAX_CTR", def.ProxyMaxCTR),
		ProxyMinUsers:              envFloatPolicy("FRAUD_POLICY_PROXY_MIN_USERS", def.ProxyMinUsers),
		ProxyMinUserClickGap:       envFloatPolicy("FRAUD_POLICY_PROXY_MIN_USER_CLICK_GAP", def.ProxyMinUserClickGap),
		ProxyMinEventsPerUser:      envFloatPolicy("FRAUD_POLICY_PROXY_MIN_EVENTS_PER_USER", def.ProxyMinEventsPerUser),
		ProxyMinImpressionPressure: envFloatPolicy("FRAUD_POLICY_PROXY_MIN_IMPRESSION_PRESSURE", def.ProxyMinImpressionPressure),
		ProxyMinUsersPerUA:         envFloatPolicy("FRAUD_POLICY_PROXY_MIN_USERS_PER_UA", def.ProxyMinUsersPerUA),
		ProxyMinClicks:             envFloatPolicy("FRAUD_POLICY_PROXY_MIN_CLICKS", def.ProxyMinClicks),
	}
}

func envFloatPolicy(key string, fallback float64) float64 {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return fallback
	}
	return v
}

const (
	mobileTierMin = 0
	mobileTierMax = 4
)

type mobileTierSnapshot struct {
	gen   uint64
	tiers map[uint32]uint8
}

type MobileTierTable struct {
	active atomic.Pointer[mobileTierSnapshot]
}

func NewMobileTierTable() *MobileTierTable {
	t := &MobileTierTable{}
	t.Publish(builtinMobileTierASNs(), 1)
	return t
}

func (t *MobileTierTable) Publish(tiers map[uint32]uint8, gen uint64) {
	if t == nil {
		return
	}
	dup := make(map[uint32]uint8, len(tiers))
	for asn, tier := range tiers {
		if asn == 0 {
			continue
		}
		dup[asn] = clampMobileTier(tier)
	}
	t.active.Store(&mobileTierSnapshot{gen: gen, tiers: dup})
}

func (t *MobileTierTable) Ready() bool {
	return t != nil && t.active.Load() != nil
}

func (t *MobileTierTable) MobileTier(asn uint32) uint8 {
	if t == nil || asn == 0 {
		return 0
	}
	snap := t.active.Load()
	if snap == nil || len(snap.tiers) == 0 {
		return 0
	}
	tier, ok := snap.tiers[asn]
	if !ok {
		return 0
	}
	return tier
}

func clampMobileTier(tier uint8) uint8 {
	if tier < mobileTierMin {
		return mobileTierMin
	}
	if tier > mobileTierMax {
		return mobileTierMax
	}
	return tier
}

func builtinMobileTierASNs() map[uint32]uint8 {
	return map[uint32]uint8{
		310410: 4,
		58453:  4,
		45400:  4,
		26615:  3,
		21928:  3,
		20057:  3,
		6167:   2,
		3215:   2,
		12479:  2,
		3209:   2,
		12956:  2,
		3320:   2,
		9808:   2,
		2856:   2,
	}
}

type mobileASNTierFeedLoader struct {
	dir     string
	refresh time.Duration
	table   *MobileTierTable
	gen     atomic.Uint64
}

func NewMobileASNTierFeedLoader(cfg *config.Config, table *MobileTierTable) *mobileASNTierFeedLoader {
	if cfg == nil || table == nil || !cfg.MobileASNTierEnabled {
		return nil
	}
	dir := cfg.MobileASNTierFeedDir
	if dir == "" {
		dir = "/var/lib/ad-event-processor/mobile-asn-tier"
	}
	refresh := cfg.MobileASNTierFeedRefresh
	if refresh <= 0 {
		refresh = 60 * time.Second
	}
	return &mobileASNTierFeedLoader{dir: dir, refresh: refresh, table: table}
}

func (l *mobileASNTierFeedLoader) Start(ctx context.Context) {
	l.refreshOnce()
	ticker := time.NewTicker(l.refresh)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			l.refreshOnce()
		}
	}
}

func (l *mobileASNTierFeedLoader) refreshOnce() {
	path := filepath.Join(l.dir, "mobile_asn_tier.txt")
	data, err := os.ReadFile(path)
	if err != nil {
		metrics.MobileASNTierFeedRefreshErrorsTotal.Inc()
		if l.table.Ready() {
			return
		}
		metrics.MobileASNTierUninitialized.Set(1)
		return
	}
	tiers := parseMobileTierFeed(data)
	if len(tiers) == 0 {
		metrics.MobileASNTierFeedRefreshErrorsTotal.Inc()
		if l.table.Ready() {
			return
		}
	}
	merged := mergeMobileTierFeeds(builtinMobileTierASNs(), tiers)
	gen := l.gen.Add(1)
	l.table.Publish(merged, gen)
	metrics.MobileASNTierFeedRefreshTotal.Inc()
	metrics.MobileASNTierEntries.Set(float64(len(merged)))
	metrics.MobileASNTierUninitialized.Set(0)
}

func mergeMobileTierFeeds(builtin, file map[uint32]uint8) map[uint32]uint8 {
	out := make(map[uint32]uint8, len(builtin)+len(file))
	for asn, tier := range builtin {
		out[asn] = tier
	}
	for asn, tier := range file {
		if asn != 0 {
			out[asn] = clampMobileTier(tier)
		}
	}
	return out
}

func parseMobileTierFeed(data []byte) map[uint32]uint8 {
	out := make(map[uint32]uint8)
	for line := range splitLineIter(data) {
		asn, tier, ok := parseMobileTierLine(line)
		if ok {
			out[asn] = tier
		}
	}
	return out
}

func parseMobileTierLine(line string) (uint32, uint8, bool) {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") {
		return 0, 0, false
	}
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return 0, 0, false
	}
	asn, ok := ParseASNLine(fields[0])
	if !ok {
		return 0, 0, false
	}
	tier64, err := strconv.ParseUint(fields[1], 10, 8)
	if err != nil {
		return 0, 0, false
	}
	return asn, clampMobileTier(uint8(tier64)), true
}

type MobileProbeRiskWeights struct {
	Alpha float64
	Beta  float64
	Gamma float64
}

func DefaultMobileProbeRiskWeights() MobileProbeRiskWeights {
	return MobileProbeRiskWeights{Alpha: 1.0, Beta: 0.5, Gamma: 0.25}
}

func MobileProbeRiskWeightsFromEnv() MobileProbeRiskWeights {
	def := DefaultMobileProbeRiskWeights()
	return MobileProbeRiskWeights{
		Alpha: envFloatPolicy("CROWD_PROBE_ASN_WEIGHT_ALPHA", def.Alpha),
		Beta:  envFloatPolicy("CROWD_PROBE_ASN_WEIGHT_BETA", def.Beta),
		Gamma: envFloatPolicy("CROWD_PROBE_ASN_WEIGHT_GAMMA", def.Gamma),
	}
}

func ComputeMobileProbeRisk(weights MobileProbeRiskWeights, asnTier uint8, probeScore uint8, clusterHistory float64) float64 {
	if clusterHistory < 0 {
		clusterHistory = 0
	}
	if clusterHistory > 1 {
		clusterHistory = 1
	}
	return weights.Alpha*float64(asnTier) +
		weights.Beta*float64(probeScore) +
		weights.Gamma*clusterHistory*100
}

func (r *ResidentialProxyRing) Peek(campaignHash uint32) (row ResidentialProxyRow, signal bool) {
	if r == nil {
		return ResidentialProxyRow{}, false
	}
	start := residentialProxySlotHash(campaignHash)
	for probe := range residentialProxyMaxProbe {
		idx := (start + uint32(probe)) & residentialProxySlotMask
		cell := &r.cells[idx]
		if cell.campaignHash.Load() != campaignHash {
			continue
		}
		row = residentialProxyRow{
			Events:      int(cell.events.Load()),
			Clicks:      int(cell.clicks.Load()),
			UniqueUsers: countDistinctHashes(cell.userHashes[:]),
			UniqueUAs:   countDistinctHashes(cell.uaHashes[:]),
		}
		signal = residentialProxySignal(row, r.policySnapshot())
		return row, signal
	}
	return ResidentialProxyRow{}, false
}

//go:linkname monotonicNano runtime.nanotime
func monotonicNano() int64
