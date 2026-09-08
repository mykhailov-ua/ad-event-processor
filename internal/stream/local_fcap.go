package stream

import (
	"sync/atomic"

	"ad-event-processor/internal/rtb"
)

const (
	localFcapCacheLine = 64
	localFcapSlotCount = 4096
	localFcapSlotMask  = localFcapSlotCount - 1
	localFcapMaxProbe  = 8
)

// LocalFcapCell is one open-addressed freq-cap slot. lookupKey tags the occupant;
// count is impressions/clicks accepted locally before async Redis INCR reconciles.
type LocalFcapCell struct {
	lookupKey uint64
	count     atomic.Uint32
	windowEnd atomic.Uint32
	_         [localFcapCacheLine - 8 - 4 - 4]byte
}

// LocalFcapLedger is the per-tracker in-memory freq-cap counter (local quanta for fcap).
// Hot path: TryAcquire before budget-fast.lua; Rollback on post-acquire Lua failure.
type LocalFcapLedger struct {
	cells [localFcapSlotCount]LocalFcapCell
}

func NewLocalFcapLedger() *LocalFcapLedger {
	return &LocalFcapLedger{}
}

func (l *LocalFcapLedger) cellFor(lookup uint64) *LocalFcapCell {
	if lookup == 0 {
		return &l.cells[0]
	}
	idx := lookup & localFcapSlotMask
	for probe := 0; probe < localFcapMaxProbe; probe++ {
		cell := &l.cells[(idx+uint64(probe))&localFcapSlotMask]
		if cell.lookupKey == lookup || cell.lookupKey == 0 {
			return cell
		}
	}
	return &l.cells[idx]
}

func (l *LocalFcapLedger) normalizeWindow(windowSec int32, nowSec uint32, cell *LocalFcapCell) {
	if windowSec <= 0 {
		windowSec = 3600
	}
	end := cell.windowEnd.Load()
	if end == 0 || nowSec >= end {
		cell.count.Store(0)
		cell.windowEnd.Store(nowSec + uint32(windowSec))
	}
}

// WouldExceed reports whether the next accept would hit the freq cap (read-only).
func (l *LocalFcapLedger) WouldExceed(lookup uint64, limit uint32, windowSec int32, nowSec uint32) bool {
	if lookup == 0 || limit == 0 {
		return false
	}
	cell := l.cellFor(lookup)
	if cell.lookupKey != 0 && cell.lookupKey != lookup {
		return false
	}
	l.normalizeWindow(windowSec, nowSec, cell)
	return rtb.FreqCapExceeded(limit, cell.count.Load())
}

// TryAcquire increments the local fcap counter when under limit; false when cap exceeded.
func (l *LocalFcapLedger) TryAcquire(lookup uint64, limit uint32, windowSec int32, nowSec uint32) bool {
	if lookup == 0 || limit == 0 {
		return true
	}
	cell := l.cellFor(lookup)
	if cell.lookupKey == 0 {
		cell.lookupKey = lookup
	} else if cell.lookupKey != lookup {
		return false
	}
	l.normalizeWindow(windowSec, nowSec, cell)
	for {
		cnt := cell.count.Load()
		if rtb.FreqCapExceeded(limit, cnt) {
			return false
		}
		if cell.count.CompareAndSwap(cnt, cnt+1) {
			return true
		}
	}
}

// Rollback decrements after Lua or enqueue failure (best-effort; no-op when count is zero).
func (l *LocalFcapLedger) Rollback(lookup uint64) {
	if lookup == 0 {
		return
	}
	cell := l.cellFor(lookup)
	if cell.lookupKey != lookup {
		return
	}
	for {
		cnt := cell.count.Load()
		if cnt == 0 {
			return
		}
		if cell.count.CompareAndSwap(cnt, cnt-1) {
			return
		}
	}
}
