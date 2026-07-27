// Package iogate serializes mmap append and fsync on shared NVMe for region-proxy and global ingest.
package iogate

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"sync/atomic"
	"time"

	"golang.org/x/sys/cpu"
)

// Tier selects append priority when the disk gate is contended or degraded.
type Tier int

const (
	TierHigh Tier = iota
	TierLow
)

// ErrShed is returned when TierLow is rejected while the gate is degraded.
var ErrShed = errors.New("disk gate shed")

const (
	DefaultDiskLatencyBudget   = 50 * time.Millisecond
	DefaultGroupCommitRecords  = 64
	DefaultGroupCommitInterval = 100 * time.Millisecond
	DefaultAppendCapacity      = 32
	envDiskLatencyBudgetMS     = "DISK_LATENCY_BUDGET_MS"
)

// Config tunes append concurrency, group-commit batching, and degraded thresholds.
type Config struct {
	AppendCapacity      int
	DiskLatencyBudget   time.Duration
	GroupCommitRecords  int64
	GroupCommitInterval time.Duration
	// DiskWritable reports whether the backing volume accepts writes; nil means writable.
	DiskWritable func() bool
}

// DefaultConfig returns production defaults aligned with MULTI_REGION §4 and M1.
func DefaultConfig() Config {
	return Config{
		AppendCapacity:      DefaultAppendCapacity,
		DiskLatencyBudget:   diskLatencyBudgetFromEnv(),
		GroupCommitRecords:  DefaultGroupCommitRecords,
		GroupCommitInterval: DefaultGroupCommitInterval,
	}
}

func diskLatencyBudgetFromEnv() time.Duration {
	raw := os.Getenv(envDiskLatencyBudgetMS)
	if raw == "" {
		return DefaultDiskLatencyBudget
	}
	ms, err := strconv.Atoi(raw)
	if err != nil || ms <= 0 {
		return DefaultDiskLatencyBudget
	}
	return time.Duration(ms) * time.Millisecond
}

// DiskWriteGate limits concurrent mmap appends and serializes fsync with EMA-based degradation.
type DiskWriteGate struct {
	appendSem chan struct{}
	fsyncSem  chan struct{}

	cfg Config

	degraded   atomic.Uint32
	emaLatency atomic.Uint64

	inFlight atomic.Int32
	_        cpu.CacheLinePad

	fsyncInFlight atomic.Int32
	pendingFsync  atomic.Int64
	lastFlushAt   atomic.Int64
}

// NewDiskWriteGate builds a gate with the configured append and fsync semaphores.
func NewDiskWriteGate(cfg Config) *DiskWriteGate {
	if cfg.AppendCapacity <= 0 {
		cfg.AppendCapacity = DefaultAppendCapacity
	}
	if cfg.DiskLatencyBudget <= 0 {
		cfg.DiskLatencyBudget = diskLatencyBudgetFromEnv()
	}
	if cfg.GroupCommitRecords <= 0 {
		cfg.GroupCommitRecords = DefaultGroupCommitRecords
	}
	if cfg.GroupCommitInterval <= 0 {
		cfg.GroupCommitInterval = DefaultGroupCommitInterval
	}
	return &DiskWriteGate{
		appendSem: make(chan struct{}, cfg.AppendCapacity),
		fsyncSem:  make(chan struct{}, 1),
		cfg:       cfg,
	}
}

// AcquireAppend blocks until an append slot is available or ctx is cancelled.
// TierLow is shed immediately when degraded or disk is not writable.
func (g *DiskWriteGate) AcquireAppend(ctx context.Context, tier Tier) error {
	if g == nil {
		return nil
	}
	if tier == TierLow && g.degraded.Load() == 1 {
		incShedTotal()
		return fmt.Errorf("disk gate acquire tier=%d: %w", tier, ErrShed)
	}
	if g.cfg.DiskWritable != nil && !g.cfg.DiskWritable() {
		g.setDegraded(1)
		if tier == TierLow {
			incShedTotal()
			return fmt.Errorf("disk gate acquire tier=%d: %w", tier, ErrShed)
		}
	}
	start := time.Now()
	select {
	case g.appendSem <- struct{}{}:
		if wait := time.Since(start); wait > 0 {
			observeAppendWait(tier, wait.Seconds())
		}
		g.inFlight.Add(1)
		return nil
	case <-ctx.Done():
		return fmt.Errorf("disk gate acquire tier=%d: %w", tier, ctx.Err())
	}
}

// ReleaseAppend returns an append slot acquired by AcquireAppend.
func (g *DiskWriteGate) ReleaseAppend(tier Tier) {
	if g == nil {
		return
	}
	g.inFlight.Add(-1)
	<-g.appendSem
	_ = tier
}

// AcquireFsync blocks until the single fsync slot is available or ctx is cancelled.
func (g *DiskWriteGate) AcquireFsync(ctx context.Context) error {
	if g == nil {
		return nil
	}
	select {
	case g.fsyncSem <- struct{}{}:
		g.fsyncInFlight.Add(1)
		setFsyncInFlight(float64(g.fsyncInFlight.Load()))
		return nil
	case <-ctx.Done():
		return fmt.Errorf("disk gate acquire fsync: %w", ctx.Err())
	}
}

// ReleaseFsync records fsync latency, updates the EMA degraded flag, and releases the fsync slot.
func (g *DiskWriteGate) ReleaseFsync(latency time.Duration) {
	if g == nil {
		return
	}
	g.recordFsyncLatency(latency)
	g.fsyncInFlight.Add(-1)
	setFsyncInFlight(float64(g.fsyncInFlight.Load()))
	<-g.fsyncSem
	g.resetGroupCommitClock()
}

// NoteAppend increments the group-commit counter and reports whether fsync should run.
func (g *DiskWriteGate) NoteAppend() bool {
	if g == nil {
		return false
	}
	n := g.pendingFsync.Add(1)
	if n >= g.cfg.GroupCommitRecords {
		g.pendingFsync.Store(0)
		g.resetGroupCommitClock()
		return true
	}
	now := time.Now().UnixNano()
	last := g.lastFlushAt.Load()
	if last == 0 {
		g.lastFlushAt.Store(now)
		return false
	}
	if time.Duration(now-last) >= g.cfg.GroupCommitInterval {
		g.pendingFsync.Store(0)
		g.resetGroupCommitClock()
		return true
	}
	return false
}

func (g *DiskWriteGate) resetGroupCommitClock() {
	g.lastFlushAt.Store(time.Now().UnixNano())
}

func (g *DiskWriteGate) recordFsyncLatency(latency time.Duration) {
	latencyNs := uint64(latency.Nanoseconds())
	var newEMA uint64
	for {
		cur := g.emaLatency.Load()
		if cur == 0 {
			newEMA = latencyNs
		} else {
			newEMA = (latencyNs + 9*cur) / 10
		}
		if g.emaLatency.CompareAndSwap(cur, newEMA) {
			break
		}
	}
	budgetNs := uint64(g.cfg.DiskLatencyBudget.Nanoseconds())
	if newEMA > budgetNs {
		g.setDegraded(1)
	}
}

func (g *DiskWriteGate) setDegraded(v uint32) {
	g.degraded.Store(v)
	setDegradedMetric(float64(v))
}

// Degraded reports whether TierLow appends are being shed.
func (g *DiskWriteGate) Degraded() bool {
	if g == nil {
		return false
	}
	return g.degraded.Load() == 1
}

// SetDegraded forces degraded state (tests and disk monitor hooks).
func (g *DiskWriteGate) SetDegraded(v bool) {
	if g == nil {
		return
	}
	var n uint32
	if v {
		n = 1
	}
	g.setDegraded(n)
}

// EMALatency returns the current fsync latency EMA in nanoseconds.
func (g *DiskWriteGate) EMALatency() time.Duration {
	if g == nil {
		return 0
	}
	return time.Duration(g.emaLatency.Load())
}

// InFlight returns holders that acquired but have not released an append slot.
func (g *DiskWriteGate) InFlight() int {
	if g == nil {
		return 0
	}
	return int(g.inFlight.Load())
}

// FsyncInFlight returns holders that acquired but have not released the fsync slot.
func (g *DiskWriteGate) FsyncInFlight() int {
	if g == nil {
		return 0
	}
	return int(g.fsyncInFlight.Load())
}

// AppendCapacity returns the configured append semaphore capacity.
func (g *DiskWriteGate) AppendCapacity() int {
	if g == nil {
		return 0
	}
	return g.cfg.AppendCapacity
}

// PendingFsyncRecords returns records not yet covered by a group-commit fsync.
func (g *DiskWriteGate) PendingFsyncRecords() int64 {
	if g == nil {
		return 0
	}
	return g.pendingFsync.Load()
}

// FlushDueByInterval reports whether the group-commit interval elapsed with pending records.
func (g *DiskWriteGate) FlushDueByInterval() bool {
	if g == nil || g.pendingFsync.Load() == 0 {
		return false
	}
	last := g.lastFlushAt.Load()
	if last == 0 {
		return false
	}
	if time.Duration(time.Now().UnixNano()-last) < g.cfg.GroupCommitInterval {
		return false
	}
	g.pendingFsync.Store(0)
	g.resetGroupCommitClock()
	return true
}
