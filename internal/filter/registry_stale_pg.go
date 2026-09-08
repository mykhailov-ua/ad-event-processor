package filter

import (
	"sync/atomic"
	"time"
)

type stalePGReadGate struct {
	maxPerSec   atomic.Int64
	windowUnix  atomic.Int64
	windowCount atomic.Uint64
}

func (g *stalePGReadGate) SetMaxPerSec(n int) {
	if g == nil {
		return
	}
	g.maxPerSec.Store(int64(n))
}

func (g *stalePGReadGate) allow(now time.Time) bool {
	if g == nil {
		return true
	}
	max := g.maxPerSec.Load()
	if max <= 0 {
		return true
	}
	sec := now.Unix()
	w := g.windowUnix.Load()
	if w != sec {
		g.windowUnix.Store(sec)
		g.windowCount.Store(0)
	}
	return g.windowCount.Add(1) <= uint64(max)
}

func (r *Registry) SetStalePGMaxRPS(maxPerSec int) {
	if r == nil {
		return
	}
	r.stalePGGate.SetMaxPerSec(maxPerSec)
}

func (r *Registry) tryAcquireStalePGRead(now time.Time) bool {
	if r == nil {
		return false
	}
	return r.stalePGGate.allow(now)
}
