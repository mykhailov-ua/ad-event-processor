package unified

import (
	"context"
	_ "embed"
	"hash/crc32"
	"hash/maphash"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"ad-event-processor/internal/database"
	"ad-event-processor/internal/domain"
	filt "ad-event-processor/internal/filter"
	"ad-event-processor/internal/metrics"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"
)

type budgetArgs struct {
	campKeyBuf [64]byte
	idemKeyBuf [64]byte
	campIDBuf  [36]byte
	custIDBuf  [36]byte
	amountBuf  [32]byte
	ttlBuf     [32]byte

	campaignKey    string
	idempotencyKey string
	campaignIDStr  string
	customerIDStr  string
	amountStr      string
	ttlStr         string

	keys [2]string
	args [4]any
}

var budgetArgsPool = sync.Pool{
	New: func() any {
		return &budgetArgs{}
	},
}

const budgetLuaScript = `
if redis.call("EXISTS", KEYS[2]) == 1 then
 return 1
end

local b = redis.call("GET", KEYS[1])
if not b then
 return -1
end

local budget = tonumber(b)
local amount = tonumber(ARGV[1])

if budget < amount then
 return 0
end

redis.call("INCRBY", KEYS[1], -amount)
redis.call("INCRBY", "budget:sync:campaign:" .. ARGV[3], ARGV[1])
redis.call("INCRBY", "budget:sync:customer:" .. ARGV[4], ARGV[1])
redis.call("SADD", "budget:dirty_campaigns", ARGV[3])
redis.call("SADD", "budget:dirty_customers", ARGV[4])
redis.call("SET", KEYS[2], "1", "EX", ARGV[2])

return 1
`

type RedisBudgetManager struct {
	redisClient    redis.Cmdable
	campaignRepo   domain.CampaignRepository
	idempotencyTTL time.Duration
}

func NewRedisBudgetManager(redisClient redis.Cmdable, repo domain.CampaignRepository, idempotencyTTL time.Duration) *RedisBudgetManager {
	return &RedisBudgetManager{
		redisClient:    redisClient,
		campaignRepo:   repo,
		idempotencyTTL: idempotencyTTL,
	}
}

func (m *RedisBudgetManager) CheckAndSpend(ctx context.Context, customerID, campaignID uuid.UUID, clickID string, amount int64) (bool, error) {
	ba := budgetArgsPool.Get().(*budgetArgs)
	defer budgetArgsPool.Put(ba)

	campKeySlice := ba.campKeyBuf[:0]
	campKeySlice = append(campKeySlice, "budget:campaign:"...)
	campKeySlice = filt.AppendUUID(campKeySlice, campaignID)
	ba.campaignKey = filt.UnsafeString(campKeySlice)

	idemKeySlice := ba.idemKeyBuf[:0]
	idemKeySlice = append(idemKeySlice, "idempotency:click:"...)
	idemKeySlice = append(idemKeySlice, clickID...)
	ba.idempotencyKey = filt.UnsafeString(idemKeySlice)

	campIDSlice := ba.campIDBuf[:0]
	campIDSlice = filt.AppendUUID(campIDSlice, campaignID)
	ba.campaignIDStr = filt.UnsafeString(campIDSlice)

	custIDSlice := ba.custIDBuf[:0]
	custIDSlice = filt.AppendUUID(custIDSlice, customerID)
	ba.customerIDStr = filt.UnsafeString(custIDSlice)

	amountSlice := ba.amountBuf[:0]
	amountSlice = strconv.AppendInt(amountSlice, amount, 10)
	ba.amountStr = filt.UnsafeString(amountSlice)

	ttlSlice := ba.ttlBuf[:0]
	ttlSlice = strconv.AppendInt(ttlSlice, int64(m.idempotencyTTL.Seconds()), 10)
	ba.ttlStr = filt.UnsafeString(ttlSlice)

	ba.keys[0] = ba.campaignKey
	ba.keys[1] = ba.idempotencyKey

	ba.args[0] = &ba.amountStr
	ba.args[1] = &ba.ttlStr
	ba.args[2] = &ba.campaignIDStr
	ba.args[3] = &ba.customerIDStr

	for i := range 2 {
		res, err := m.redisClient.Eval(ctx, budgetLuaScript, ba.keys[:], ba.args[:]...).Int64()
		if err != nil {
			return false, err
		}

		if res == -1 {
			if i > 0 {
				return false, filt.ErrBudgetExhausted
			}

			camp, err := m.campaignRepo.GetByID(ctx, campaignID)
			if err != nil {
				return false, err
			}

			remaining := camp.BudgetLimit - camp.CurrentSpend
			if remaining < 0 {
				remaining = 0
			}

			m.redisClient.SetNX(ctx, ba.campaignKey, remaining, 24*time.Hour)
			continue
		}

		return res == 1, nil
	}

	return false, filt.ErrBudgetExhausted
}

//go:embed ip-rate-limit.lua
var ipRateLimitLua string

var (
	ipRateLimitLuaAny = ipRateLimitLua
	ipRateLimitScript = redis.NewScript(ipRateLimitLua)
)

type IPRateLimiter struct {
	redisClient   redis.UniversalClient
	limit         int
	scriptHashAny any
	scriptAny     any
	windowMsAny   any
	wire          [5]any
}

func NewIPRateLimiter(redisClient redis.UniversalClient, limit int, window time.Duration) *IPRateLimiter {
	ms := window.Milliseconds()
	l := &IPRateLimiter{
		redisClient:   redisClient,
		limit:         limit,
		scriptHashAny: ipRateLimitScript.Hash(),
		scriptAny:     ipRateLimitLuaAny,
		windowMsAny:   ms,
	}
	l.wire[0] = evalShaCmdAny
	l.wire[2] = numKeys1Any
	return l
}

func (l *IPRateLimiter) Check(ctx context.Context, evt *domain.Event) error {
	if evt.IP == "" {
		return nil
	}

	w := filt.AcquireBufWrapper()
	w.Buf = w.Buf[:0]
	w.Buf = append(w.Buf, "ratelimit:ip:"...)
	w.Buf = append(w.Buf, evt.IP...)
	key := filt.UnsafeString(w.Buf)

	count, err := l.evalRateLimit(ctx, key)
	filt.ReleaseBufWrapper(w)
	if err != nil {
		return err
	}
	if count > int64(l.limit) {
		return filt.ErrRateLimitExceeded
	}
	return nil
}

func (l *IPRateLimiter) evalRateLimit(ctx context.Context, key string) (int64, error) {
	l.wire[1] = l.scriptHashAny
	l.wire[3] = key
	l.wire[4] = l.windowMsAny

	cmd := evalCmdPool.Get().(*redis.Cmd)
	resetPooledRedisCmd(cmd, ctx, l.wire[:], 3)
	err := l.redisClient.Process(ctx, cmd)
	val, intErr := cmd.Int64()
	if intErr != nil && err == nil {
		err = intErr
	}
	evalCmdPool.Put(cmd)
	if err != nil && isNoScriptErr(err) {
		return l.evalRateLimitScript(ctx, key)
	}
	if err != nil {
		return 0, err
	}
	return val, nil
}

func (l *IPRateLimiter) evalRateLimitScript(ctx context.Context, key string) (int64, error) {
	l.wire[0] = evalCmdAny
	l.wire[1] = l.scriptAny
	l.wire[3] = key
	l.wire[4] = l.windowMsAny

	cmd := evalCmdPool.Get().(*redis.Cmd)
	resetPooledRedisCmd(cmd, ctx, l.wire[:], 3)
	err := l.redisClient.Process(ctx, cmd)
	val, intErr := cmd.Int64()
	if intErr != nil && err == nil {
		err = intErr
	}
	evalCmdPool.Put(cmd)
	l.wire[0] = evalShaCmdAny
	l.wire[1] = l.scriptHashAny
	if err != nil {
		return 0, err
	}
	return val, nil
}

func debitSubSlot(camp *domain.Campaign, userID, clickID string) int {
	n := camp.DebitSubShardCount()
	if n <= 1 {
		return 0
	}
	var key []byte
	if userID != "" {
		key = filt.UnsafeBytes(userID)
	} else if clickID != "" {
		key = filt.UnsafeBytes(clickID)
	}
	if len(key) == 0 {
		return 0
	}
	h := ComputeCompositeHashUUID(camp.ID, key)
	return int(h % uint32(n))
}

func DebitSubSlot(camp *domain.Campaign, userID, clickID string) int {
	return debitSubSlot(camp, userID, clickID)
}

func spreadHighVolumeShard(shardCount int, campaignID uuid.UUID, subSlot int) int {
	if shardCount <= 1 {
		return 0
	}
	h := domain.CRC32Castagnoli(&campaignID) + uint32(subSlot)
	return int(h % uint32(shardCount))
}

func appendBudgetQuotaKey(dst []byte, campaignID uuid.UUID, subSlot int) []byte {
	if subSlot <= 0 {
		dst = filt.AppendCampaignHashTag(dst, campaignID)
	} else {
		dst = domain.AppendCampaignSubHashTag(dst, campaignID, subSlot)
	}
	dst = append(dst, "budget:quota:"...)
	return filt.AppendUUID(dst, campaignID)
}

func budgetQuotaKeyForDebit(campaignID uuid.UUID, subSlot int) string {
	if subSlot <= 0 {
		return filt.BudgetQuotaKey(campaignID)
	}
	return domain.BudgetQuotaKeySub(campaignID, subSlot)
}

func FcapKeyPrefixForDebit(camp *domain.Campaign, userID, clickID string) string {
	return fcapKeyPrefixForDebit(camp, userID, clickID)
}

func fcapKeyPrefixForDebit(camp *domain.Campaign, userID, clickID string) string {
	if camp == nil {
		return ""
	}
	sub := debitSubSlot(camp, userID, clickID)
	return domain.FcapKeyPrefixSub(camp.ID, camp.BrandFcapKey, sub)
}

const localTTCCapacity = 65536

type localTTCSlot struct {
	campaignID uuid.UUID
	userHash   uint64
	impTSMs    int64
	valid      bool
}

type LocalTTCCache struct {
	seed  maphash.Seed
	mu    sync.RWMutex
	slots [localTTCCapacity]localTTCSlot
}

func NewLocalTTCCache() *LocalTTCCache {
	return &LocalTTCCache{seed: maphash.MakeSeed()}
}

func (c *LocalTTCCache) userHash(campaignID uuid.UUID, userID string) uint64 {
	var h maphash.Hash
	h.SetSeed(c.seed)
	h.Write(campaignID[:])
	h.WriteString(userID)
	return h.Sum64()
}

func (c *LocalTTCCache) Record(campaignID uuid.UUID, userID string) {
	if userID == "" {
		return
	}
	uh := c.userHash(campaignID, userID)
	idx := uh % localTTCCapacity
	nowMs := time.Now().UnixMilli()
	c.mu.Lock()
	c.slots[idx] = localTTCSlot{
		campaignID: campaignID,
		userHash:   uh,
		impTSMs:    nowMs,
		valid:      true,
	}
	c.mu.Unlock()
}

type LocalTTCOutcome int

const (
	LocalTTCOK LocalTTCOutcome = iota
	LocalTTCLow
	LocalTTCMissingClosed
	LocalTTCBypass
)

const (
	localTTCOK            = LocalTTCOK
	localTTCLow           = LocalTTCLow
	localTTCMissingClosed = LocalTTCMissingClosed
	localTTCBypass        = LocalTTCBypass
)

type localTTCOutcome = LocalTTCOutcome

func (c *LocalTTCCache) CheckClick(campaignID uuid.UUID, userID string, minMs int64, failClosed bool) localTTCOutcome {
	if userID == "" {
		if failClosed {
			return localTTCMissingClosed
		}
		return localTTCBypass
	}
	uh := c.userHash(campaignID, userID)
	idx := uh % localTTCCapacity
	c.mu.RLock()
	slot := c.slots[idx]
	c.mu.RUnlock()
	if !slot.valid || slot.campaignID != campaignID || slot.userHash != uh {
		if failClosed {
			return localTTCMissingClosed
		}
		return localTTCBypass
	}
	if time.Now().UnixMilli()-slot.impTSMs < minMs {
		return localTTCLow
	}
	return localTTCOK
}

func ttcMinMs(ttcMinMsAny any) int64 {
	switch v := ttcMinMsAny.(type) {
	case int64:
		return v
	case int:
		return int64(v)
	default:
		return 0
	}
}

func (f *UnifiedFilter) SetLocalTTCCache(c *LocalTTCCache) {
	f.localTTC = c
}

func (f *UnifiedFilter) LocalTTC() *LocalTTCCache {
	return f.localTTC
}

func (f *UnifiedFilter) ApplyGoTTC(evt *domain.Event) {
	f.applyGoTTC(evt)
}

func (f *UnifiedFilter) applyGoTTC(evt *domain.Event) {
	if f == nil || f.localTTC == nil || evt == nil || !ttcEnabled(f.ttcMinMsAny) {
		return
	}
	minMs := ttcMinMs(f.ttcMinMsAny)
	if evt.Type == "impression" && evt.UserID != "" {
		f.localTTC.Record(evt.CampaignID, evt.UserID)
		return
	}
	if evt.Type != "click" {
		return
	}
	failClosed := f.ttcFailClosedAny == oneAny
	if !failClosed && f.registry != nil {
		if camp, ok := f.getCampaign(evt); ok && camp != nil {
			if domain.ResolveAttestationMode(camp.AttestationMode, camp.AttestationEnabled).RequiresProbe() {
				failClosed = true
			}
		}
	}
	switch f.localTTC.CheckClick(evt.CampaignID, evt.UserID, minMs, failClosed) {
	case localTTCLow:
		filt.AddFraudSignal(evt, filt.FraudReasonLowTTC)
	case localTTCMissingClosed:
		filt.AddFraudSignal(evt, filt.FraudReasonMissingImpTS)
	case localTTCBypass:
		metrics.TTCBypassTotal.Inc()
	}
}

const roughPacingCells = 4096

type roughPacingCell struct {
	campaignID uuid.UUID
	dayKey     uint32
	spentMicro atomic.Int64
}

type RoughPacingGate struct {
	seed  maphash.Seed
	mu    sync.Mutex
	cells [roughPacingCells]roughPacingCell
}

func NewRoughPacingGate() *RoughPacingGate {
	return &RoughPacingGate{seed: maphash.MakeSeed()}
}

func (g *RoughPacingGate) dayKey(t time.Time) uint32 {
	y, m, d := t.Date()
	return uint32(y)*10000 + uint32(m)*100 + uint32(d)
}

func (g *RoughPacingGate) cellFor(campaignID uuid.UUID, day uint32) *roughPacingCell {
	var h maphash.Hash
	h.SetSeed(g.seed)
	h.Write(campaignID[:])
	idx := h.Sum64() % roughPacingCells
	cell := &g.cells[idx]
	g.mu.Lock()
	if cell.campaignID != campaignID || cell.dayKey != day {
		cell.campaignID = campaignID
		cell.dayKey = day
		cell.spentMicro.Store(0)
	}
	g.mu.Unlock()
	return cell
}

func (g *RoughPacingGate) Allow(campaignID uuid.UUID, amountMicro, dailyBudgetMicro int64, hour int) bool {
	if g == nil || dailyBudgetMicro <= 0 || amountMicro <= 0 {
		return true
	}
	if hour < 1 {
		hour = 1
	} else if hour > 24 {
		hour = 24
	}
	cumulativeLimit := (dailyBudgetMicro * int64(hour)) / 24
	now := filt.CachedTimeUTC()
	cell := g.cellFor(campaignID, g.dayKey(now))
	spent := cell.spentMicro.Add(amountMicro)
	if spent > cumulativeLimit {
		cell.spentMicro.Add(-amountMicro)
		return false
	}
	return true
}

func (f *UnifiedFilter) SetRoughPacingGate(g *RoughPacingGate) {
	f.roughPacing = g
}

func (f *UnifiedFilter) checkGoRoughPacing(evt *domain.Event, camp *domain.Campaign, amountMicro int64) error {
	if f == nil || f.roughPacing == nil || camp == nil || !camp.RoughPacingEnabled() {
		return nil
	}
	if camp.PacingMode != domain.PacingModeEven {
		return nil
	}
	if evt.Type != "impression" && evt.Type != "click" {
		return nil
	}
	hr := filt.CachedTimeUTC().Hour() + 1
	if !f.roughPacing.Allow(camp.ID, amountMicro, camp.DailyBudgetMicro, hr) {
		return filt.ErrPacingExhausted
	}
	return nil
}

func indexByteString(s string, b byte) int {
	for i := range len(s) {
		if s[i] == b {
			return i
		}
	}
	return -1
}

func loadU64(b []byte) uint64 {
	return *(*uint64)(unsafe.Pointer(&b[0]))
}

func newRedisLuaObservers(numShards int) []prometheus.Observer {
	if numShards <= 0 {
		numShards = 1
	}
	observers := make([]prometheus.Observer, numShards)
	for i := range observers {
		observers[i] = metrics.RedisLuaDuration.WithLabelValues(strconv.Itoa(i))
	}
	return observers
}

func newRedisLuaNoScriptCounters(numShards int) []prometheus.Counter {
	if numShards <= 0 {
		numShards = 1
	}
	counters := make([]prometheus.Counter, numShards)
	for i := range counters {
		counters[i] = metrics.RedisLuaNoScriptTotal.WithLabelValues(strconv.Itoa(i))
	}
	return counters
}

func (f *UnifiedFilter) resolveDebitShard(campaignID uuid.UUID, userID, clickID string, campInfo *domain.Campaign) (shard int, subSlot int, err error) {
	shard = f.sharder.GetShard(campaignID)
	subSlot = 0

	if campInfo != nil && campInfo.DebitSubShardCount() > 0 {
		subSlot = debitSubSlot(campInfo, userID, clickID)
		shard = spreadHighVolumeShard(len(f.redisShards), campaignID, subSlot)
	} else if campInfo != nil && campInfo.HasTriplet {
		hash := ComputeCompositeHashUUID(campaignID, []byte(userID))
		pct := hash % 100
		switch {
		case pct < 40:
			shard = int(campInfo.PrimaryAShard)
		case pct < 80:
			shard = int(campInfo.PrimaryBShard)
		default:
			shard = int(campInfo.ReserveShard)
		}
	}

	if !f.shardBreakerOpen(shard) {
		return shard, subSlot, nil
	}

	if campInfo != nil && campInfo.HasTriplet && campInfo.DebitSubShardCount() == 0 {
		alts := [...]int{
			int(campInfo.ReserveShard),
			int(campInfo.PrimaryAShard),
			int(campInfo.PrimaryBShard),
		}
		for _, alt := range alts {
			if alt == shard {
				continue
			}
			if !f.shardBreakerOpen(alt) {
				return alt, subSlot, nil
			}
		}
	}
	return 0, 0, filt.ErrShardUnavailable
}

func (f *UnifiedFilter) shardBreakerOpen(shard int) bool {
	if len(f.breakers) == 0 {
		return false
	}
	n := len(f.breakers)
	if n == 0 {
		return false
	}
	idx := shard % n
	if idx < 0 {
		idx = -idx
	}
	b := f.breakers[idx]
	if b == nil {
		return false
	}
	return b.State() == database.CircuitOpen
}

func (f *UnifiedFilter) SetShardBreakers(breakers []*database.RedisBreaker) {
	if f == nil {
		return
	}
	f.breakers = breakers
}

func ComputeCompositeHashUUID(campaignID uuid.UUID, userID []byte) uint32 {
	var crc uint32
	var started bool
	if campaignID != uuid.Nil {
		b := filt.AppendUUID(make([]byte, 0, 36), campaignID)
		crc = crc32.ChecksumIEEE(b)
		started = true
	}
	if len(userID) > 0 {
		if started {
			crc = crc32.Update(crc, crc32.IEEETable, userID)
		} else {
			crc = crc32.ChecksumIEEE(userID)
		}
	}
	if !started && len(userID) == 0 {
		return 0
	}
	return crc
}

func hexByte(n byte) byte {
	return filt.HexByte(n)
}
