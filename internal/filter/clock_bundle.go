package filter

import (
	"os"
	"sync/atomic"
	"time"
	_ "unsafe"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
)

var nodeID uint16

var idSequence uint64

var cachedUnixMilli atomic.Int64

var cachedUnixMilliAny atomic.Value

var cachedNowUTC atomic.Pointer[time.Time]

var clockRefreshPaused atomic.Bool

func SetClockRefreshPaused(paused bool) {
	clockRefreshPaused.Store(paused)
}

func storeCachedNowUTC() {
	t := time.Now().UTC()
	cachedNowUTC.Store(&t)
}

func RefreshCachedWallClockNow() {
	ms := time.Now().UnixMilli()
	cachedUnixMilli.Store(ms)
	cachedUnixMilliAny.Store(ms)
	storeCachedNowUTC()
}

func CachedTimeUTC() time.Time {
	if p := cachedNowUTC.Load(); p != nil {
		return *p
	}
	return time.Now().UTC()
}

func CachedTimeIn(loc *time.Location) time.Time {
	if loc == nil || loc == time.UTC {
		return CachedTimeUTC()
	}
	return CachedTimeUTC().In(loc)
}

func CachedUnixMilliAnyLoad() any {
	return cachedUnixMilliAny.Load()
}

func CachedUnixMilliAnyStore(v any) {
	cachedUnixMilliAny.Store(v)
}

func CachedUnixMilliNow() int64 {
	return cachedUnixMilli.Load()
}

func CachedUnixMilliStore(ms int64) {
	cachedUnixMilli.Store(ms)
}

func StoreCachedNowUTC() {
	storeCachedNowUTC()
}

func CachedNowUTCSetFromUnixMilli(ms int64) {
	t := time.UnixMilli(ms).UTC()
	cachedNowUTC.Store(&t)
}

func CachedUnixSec() uint64 {
	return uint64(cachedUnixMilli.Load() / 1000)
}

func init() {
	hostname, _ := os.Hostname()
	h := uint32(os.Getpid())
	for _, c := range hostname {
		h = h*31 + uint32(c)
	}
	nodeID = uint16(h ^ (h >> 16))

	cachedUnixMilli.Store(time.Now().UnixMilli())
	cachedUnixMilliAny.Store(cachedUnixMilli.Load())
	storeCachedNowUTC()
	go func() {
		ticker := time.NewTicker(time.Millisecond)
		defer ticker.Stop()
		for range ticker.C {
			if clockRefreshPaused.Load() {
				continue
			}
			if until := clockTickPausedUntil.Load(); until > 0 && monotonicNano() < until {
				continue
			}
			ms := time.Now().UnixMilli()
			cachedUnixMilli.Store(ms)
			cachedUnixMilliAny.Store(ms)
		}
	}()
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for range ticker.C {
			if clockRefreshPaused.Load() {
				continue
			}
			storeCachedNowUTC()
		}
	}()
}

func NewFastUUID() uuid.UUID {
	seq := atomic.AddUint64(&idSequence, 1)
	now := cachedUnixMilli.Load()

	var u uuid.UUID

	u[0] = byte(now >> 40)
	u[1] = byte(now >> 32)
	u[2] = byte(now >> 24)
	u[3] = byte(now >> 16)
	u[4] = byte(now >> 8)
	u[5] = byte(now)

	u[6] = byte(seq >> 48)
	u[7] = byte(seq >> 40)

	u[8] = byte(nodeID >> 8)
	u[9] = byte(nodeID)

	u[10] = byte(seq >> 40)
	u[11] = byte(seq >> 32)
	u[12] = byte(seq >> 24)
	u[13] = byte(seq >> 16)
	u[14] = byte(seq >> 8)
	u[15] = byte(seq)

	u[6] = (u[6] & 0x0f) | 0x70
	u[8] = (u[8] & 0x3f) | 0x80

	return u
}

const udpCoarseTimeClampMs = 50

var clockTickPausedUntil atomic.Int64

func ApplyUDPCoarseTime(coarseTimeNs int64) {
	if coarseTimeNs <= 0 {
		return
	}
	remoteMs := coarseTimeNs / int64(time.Millisecond)
	localMs := cachedUnixMilli.Load()
	deltaMs := remoteMs - localMs
	if deltaMs > udpCoarseTimeClampMs {
		deltaMs = udpCoarseTimeClampMs
	} else if deltaMs < -udpCoarseTimeClampMs {
		deltaMs = -udpCoarseTimeClampMs
	}
	targetMs := localMs + deltaMs
	if targetMs < localMs {
		behindMs := localMs - targetMs
		clockTickPausedUntil.Store(monotonicNano() + behindMs*int64(time.Millisecond))
		return
	}
	if targetMs > localMs {
		cachedUnixMilli.Store(targetMs)
		cachedUnixMilliAny.Store(targetMs)
		t := time.UnixMilli(targetMs).UTC()
		cachedNowUTC.Store(&t)
		clockTickPausedUntil.Store(0)
	}
}

//go:linkname monotonicNano runtime.nanotime
func monotonicNano() int64

func MonotonicNano() int64 {
	return monotonicNano()
}

const nanosPerSecond = 1_000_000_000

func MonoElapsedSeconds(start int64) float64 {
	return float64(monotonicNano()-start) / nanosPerSecond
}

const (
	luaMetricsSampleMask uint64 = 127
	LuaMetricsSampleMask        = luaMetricsSampleMask
)

func HistogramSampleMaskFromConfig(cfgVal int) uint64 {
	if cfgVal < 0 {
		return 0
	}
	if cfgVal == 0 {
		return luaMetricsSampleMask
	}
	return uint64(cfgVal)
}

func ShouldSampleHistogram(seq uint64, mask uint64) bool {
	if mask == 0 {
		return true
	}
	return seq&mask == 0
}

func ShouldSampleLuaMetrics(seq uint64) bool {
	return shouldSampleLuaMetrics(seq)
}

func shouldSampleLuaMetrics(seq uint64) bool {
	return ShouldSampleHistogram(seq, luaMetricsSampleMask)
}

func ObserveHistogramSampled(seq *atomic.Uint64, mask uint64, observer prometheus.Observer, startMono int64) {
	if observer == nil {
		return
	}
	if ShouldSampleHistogram(seq.Add(1), mask) {
		observer.Observe(MonoElapsedSeconds(startMono))
	}
}

func cachedUnixSec() uint64 { return CachedUnixSec() }

func cachedUnixMilliAnyLoad() any { return CachedUnixMilliAnyLoad() }
