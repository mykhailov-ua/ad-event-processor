package ingestion

import (
	"context"
	"sync/atomic"
	"time"

	"espx/internal/metrics"
)

// ProcessorPgReserve leaves headroom on pgxpool for partition manager and health probes.
const ProcessorPgReserve = 1

// ProcessorChReserve leaves headroom on the ClickHouse connection pool for health probes.
const ProcessorChReserve = 1

// ProcessorWriteGate is a process-wide write semaphore shared by stores and SyncWorkers.
type ProcessorWriteGate struct {
	sem      chan struct{}
	capacity int
	inFlight atomic.Int32
	waitEMA  atomic.Uint64
	backend  string
}

// ProcessorPgGate is the Postgres processor write gate (SEM-P1/P2).
type ProcessorPgGate = ProcessorWriteGate

// ProcessorChGate is the ClickHouse processor write gate (SEM-P5 / stream backpressure).
type ProcessorChGate = ProcessorWriteGate

// NewProcessorPgGate sizes the gate from slots or DB_PROCESSOR_MAX_CONNS minus reserve.
// slots <= 0 selects auto mode: maxConns - ProcessorPgReserve (current default behavior).
func NewProcessorPgGate(slots, maxConns int) *ProcessorPgGate {
	return newProcessorWriteGate("postgres", slots, maxConns, ProcessorPgReserve)
}

// NewProcessorChGate sizes the gate from slots or CH_MAX_CONNS minus reserve.
// slots <= 0 selects auto mode: maxConns - ProcessorChReserve.
func NewProcessorChGate(slots, maxConns int) *ProcessorChGate {
	return newProcessorWriteGate("clickhouse", slots, maxConns, ProcessorChReserve)
}

func newProcessorWriteGate(backend string, slots, maxConns, reserve int) *ProcessorWriteGate {
	budget := slots
	if budget <= 0 {
		budget = maxConns - reserve
	}
	if budget < 1 {
		budget = 1
	}
	return &ProcessorWriteGate{
		sem:      make(chan struct{}, budget),
		capacity: budget,
		backend:  backend,
	}
}

// Acquire blocks until a write slot is available or ctx is cancelled.
func (g *ProcessorWriteGate) Acquire(ctx context.Context) error {
	if g == nil {
		return nil
	}
	start := time.Now()
	select {
	case g.sem <- struct{}{}:
		if wait := time.Since(start); wait > 0 {
			metrics.ProcessorWriteAcquireWaitSeconds.WithLabelValues(g.backend).Observe(wait.Seconds())
			g.recordWait(wait)
		}
		g.inFlight.Add(1)
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Release returns a slot acquired by Acquire.
func (g *ProcessorWriteGate) Release() {
	if g == nil {
		return
	}
	g.inFlight.Add(-1)
	<-g.sem
}

// Capacity returns the maximum concurrent writers allowed by this gate.
func (g *ProcessorWriteGate) Capacity() int {
	if g == nil {
		return 0
	}
	return g.capacity
}

// InFlight returns the number of holders that have acquired but not released.
func (g *ProcessorWriteGate) InFlight() int {
	if g == nil {
		return 0
	}
	return int(g.inFlight.Load())
}

// WaitEMA returns an exponential moving average of recent acquire waits.
func (g *ProcessorWriteGate) WaitEMA() time.Duration {
	if g == nil {
		return 0
	}
	return time.Duration(g.waitEMA.Load())
}

func (g *ProcessorWriteGate) recordWait(wait time.Duration) {
	if g == nil || wait <= 0 {
		return
	}
	for {
		old := g.waitEMA.Load()
		var next uint64
		if old == 0 {
			next = uint64(wait)
		} else {
			ema := time.Duration(old)
			ema = time.Duration(0.8*float64(ema) + 0.2*float64(wait))
			next = uint64(ema)
		}
		if g.waitEMA.CompareAndSwap(old, next) {
			return
		}
	}
}
