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

type Tier int

const (
	TierHigh Tier = iota // WAL/broker mmap: fail-open while degraded; still waits on appendSem
	TierLow              // forward/dedup: shed with ErrShed when degraded or disk not writable
)

var ErrShed = errors.New("disk gate shed")

const (
	DefaultDiskLatencyBudget   = 50 * time.Millisecond  // fsync EMA above this sets degraded
	DefaultGroupCommitRecords  = 64                     // NoteAppend fsync trigger (record count)
	DefaultGroupCommitInterval = 100 * time.Millisecond // NoteAppend fsync trigger (wall clock)
	DefaultAppendCapacity      = 32                     // appendSem buffered capacity
	envDiskLatencyBudgetMS     = "DISK_LATENCY_BUDGET_MS"
)

type Config struct {
	AppendCapacity      int
	DiskLatencyBudget   time.Duration
	GroupCommitRecords  int64
	GroupCommitInterval time.Duration
	DiskWritable        func() bool
}

func DefaultConfig() Config {
	return Config{
		AppendCapacity:      DefaultAppendCapacity,
		DiskLatencyBudget:   diskLatencyBudgetFromEnv(),
		GroupCommitRecords:  DefaultGroupCommitRecords,
		GroupCommitInterval: DefaultGroupCommitInterval,
	}
}

func TestGateConfig() Config {
	return Config{
		AppendCapacity:      256,
		DiskLatencyBudget:   time.Hour,
		GroupCommitRecords:  64,
		GroupCommitInterval: 25 * time.Millisecond,
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

// DiskWriteGate batches append concurrency and fsync on broker WAL / mmap paths.
// Callers bracket write(2)/writev(2) with AcquireAppend/ReleaseAppend and fsync(2)
// with AcquireFsync/ReleaseFsync; degraded is driven by fsync EMA vs DiskLatencyBudget.
type DiskWriteGate struct {
	appendSem chan struct{} // caps concurrent write syscalls at AppendCapacity
	fsyncSem  chan struct{} // serializes fsync/fdatasync (buffered cap 1)

	cfg Config

	degraded   atomic.Uint32  // 1 after EMA exceeds budget or DiskWritable false; sticky until SetDegraded(false)
	emaLatency atomic.Uint64 // fsync latency EMA in nanoseconds (alpha 0.1 in recordFsyncLatency)

	inFlight atomic.Int32
	_        cpu.CacheLinePad // pad inFlight from fsyncInFlight (false-sharing)

	fsyncInFlight atomic.Int32
	pendingFsync  atomic.Int64
	lastFlushAt   atomic.Int64
}

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

func (g *DiskWriteGate) AcquireAppend(ctx context.Context, tier Tier) error {
	if g == nil {
		return nil
	}
	// TierLow shed; TierHigh fail-open and proceeds to appendSem.
	if tier == TierLow && g.degraded.Load() == 1 {
		incShedTotal()
		return fmt.Errorf("disk gate acquire tier=%d: %w", tier, ErrShed)
	}
	// DiskWritable is the disk-health probe boundary (e.g. statfs, ENOSPC); false sets degraded immediately.
	if g.cfg.DiskWritable != nil && !g.cfg.DiskWritable() {
		g.setDegraded(1)
		if tier == TierLow { // TierHigh still acquires append slot
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

func (g *DiskWriteGate) ReleaseAppend(tier Tier) {
	if g == nil {
		return
	}
	g.inFlight.Add(-1)
	<-g.appendSem
	_ = tier
}

// AcquireFsync blocks until the single fsync slot is free; pair with ReleaseFsync after syscall returns.
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

func (g *DiskWriteGate) ReleaseFsync(latency time.Duration) {
	if g == nil {
		return
	}
	g.recordFsyncLatency(latency) // EMA may set degraded; affects subsequent TierLow AcquireAppend
	g.fsyncInFlight.Add(-1)
	setFsyncInFlight(float64(g.fsyncInFlight.Load()))
	<-g.fsyncSem
	g.resetGroupCommitClock()
}

func (g *DiskWriteGate) NoteAppend() bool {
	if g == nil {
		return false
	}
	n := g.pendingFsync.Add(1)
	if n >= g.cfg.GroupCommitRecords { // group-commit record threshold
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
	if time.Duration(now-last) >= g.cfg.GroupCommitInterval { // group-commit interval threshold
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
			newEMA = (latencyNs + 9*cur) / 10 // EMA alpha=0.1 (ns)
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

func (g *DiskWriteGate) Degraded() bool {
	if g == nil {
		return false
	}
	return g.degraded.Load() == 1
}

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

func (g *DiskWriteGate) EMALatency() time.Duration {
	if g == nil {
		return 0
	}
	return time.Duration(g.emaLatency.Load())
}

func (g *DiskWriteGate) InFlight() int {
	if g == nil {
		return 0
	}
	return int(g.inFlight.Load())
}

func (g *DiskWriteGate) FsyncInFlight() int {
	if g == nil {
		return 0
	}
	return int(g.fsyncInFlight.Load())
}

func (g *DiskWriteGate) AppendCapacity() int {
	if g == nil {
		return 0
	}
	return g.cfg.AppendCapacity
}

func (g *DiskWriteGate) PendingFsyncRecords() int64 {
	if g == nil {
		return 0
	}
	return g.pendingFsync.Load()
}

// FlushDueByInterval is the shutdown/drain hook: pending records past GroupCommitInterval without NoteAppend trip.
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
