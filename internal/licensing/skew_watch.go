package licensing

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

const defaultSkewWatchThreshold = 5 * time.Minute

type SkewWatch struct {
	threshold  time.Duration
	anchorWall time.Time
	anchorMono time.Duration
	lastWall   time.Time
	violated   atomic.Uint32
}

type SkewWatchOptions struct {
	Enabled   bool
	Interval  time.Duration
	Threshold time.Duration
}

var (
	skewWatchMu      sync.Mutex
	skewWatchGlobal  *SkewWatch
	skewWatchOpts    SkewWatchOptions
	skewWatchRunning atomic.Uint32
	sampleClockFn    = sampleClock
)

func wallTime(t time.Time) time.Time {
	return time.Unix(0, t.UnixNano()).UTC()
}

func sampleClock() (time.Time, time.Duration) {
	now := time.Now()
	return wallTime(now), monotonicRaw()
}

func NewSkewWatch(threshold time.Duration) *SkewWatch {
	if threshold <= 0 {
		threshold = defaultSkewWatchThreshold
	}
	wall, mono := sampleClockFn()
	return &SkewWatch{
		threshold:  threshold,
		anchorWall: wall,
		anchorMono: mono,
		lastWall:   wall,
	}
}

func (w *SkewWatch) Violated() bool {
	return w != nil && w.violated.Load() == 1
}

func (w *SkewWatch) markViolated() {
	if w != nil {
		w.violated.Store(1)
	}
}

func (w *SkewWatch) Check(wall time.Time, mono time.Duration) bool {
	if w == nil {
		return false
	}
	wall = wallTime(wall)

	if !w.lastWall.IsZero() && wall.Before(w.lastWall) {
		w.markViolated()
		return true
	}

	wallDelta := wall.Sub(w.anchorWall)
	monoDelta := mono - w.anchorMono
	if monoDelta < 0 {
		monoDelta = 0
	}
	if wallDelta < 0 {
		wallDelta = 0
	}

	lag := monoDelta - wallDelta
	lead := wallDelta - monoDelta
	if lag > w.threshold || lead > w.threshold {
		w.markViolated()
		return true
	}

	w.lastWall = wall
	w.anchorWall = wall
	w.anchorMono = mono
	return false
}

func ConfigureSkewWatch(opts SkewWatchOptions) {
	skewWatchMu.Lock()
	defer skewWatchMu.Unlock()
	if opts.Threshold <= 0 {
		opts.Threshold = defaultSkewWatchThreshold
	}
	skewWatchOpts = opts
	if !opts.Enabled {
		skewWatchGlobal = nil
		return
	}
	if skewWatchGlobal == nil {
		skewWatchGlobal = NewSkewWatch(opts.Threshold)
		return
	}
	skewWatchGlobal.threshold = opts.Threshold
}

func EvaluateClockSkew() bool {
	skewWatchMu.Lock()
	w := skewWatchGlobal
	skewWatchMu.Unlock()
	if w == nil {
		return false
	}
	wall, mono := sampleClockFn()
	return w.Check(wall, mono)
}

func ClockSkewViolated() bool {
	skewWatchMu.Lock()
	w := skewWatchGlobal
	skewWatchMu.Unlock()
	return w != nil && w.Violated()
}

func StartSkewWatch(ctx context.Context) {
	skewWatchMu.Lock()
	opts := skewWatchOpts
	skewWatchMu.Unlock()
	if !opts.Enabled || opts.Interval <= 0 {
		return
	}
	if !skewWatchRunning.CompareAndSwap(0, 1) {
		return
	}
	go func() {
		ticker := time.NewTicker(opts.Interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				EvaluateClockSkew()
			}
		}
	}()
}

func SetClockSampleHookForTest(fn func() (time.Time, time.Duration)) func() {
	prev := sampleClockFn
	if fn == nil {
		sampleClockFn = sampleClock
	} else {
		sampleClockFn = fn
	}
	return func() { sampleClockFn = prev }
}

func ResetSkewWatchForTest() {
	skewWatchMu.Lock()
	defer skewWatchMu.Unlock()
	skewWatchGlobal = nil
	skewWatchOpts = SkewWatchOptions{}
	skewWatchRunning.Store(0)
	sampleClockFn = sampleClock
}
