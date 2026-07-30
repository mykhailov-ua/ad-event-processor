package ingestion

import (
	"os"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
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
