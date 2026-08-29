package stream

import (
	"context"
	"sync/atomic"
	"time"

	"ad-event-processor/internal/metrics"
)

const ProcessorPgReserve = 1

const ProcessorChReserve = 1

type ProcessorWriteGate struct {
	sem      chan struct{}
	capacity int
	inFlight atomic.Int32
	waitEMA  atomic.Uint64
	backend  string
}

type ProcessorPostgresGate = ProcessorWriteGate

type ProcessorClickHouseGate = ProcessorWriteGate

func NewProcessorPostgresGate(slots, maxConns int) *ProcessorPostgresGate {
	return newProcessorWriteGate("postgres", slots, maxConns, ProcessorPgReserve)
}

func NewProcessorClickHouseGate(slots, maxConns int) *ProcessorClickHouseGate {
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

func (g *ProcessorWriteGate) Release() {
	if g == nil {
		return
	}
	g.inFlight.Add(-1)
	<-g.sem
}

func (g *ProcessorWriteGate) Capacity() int {
	if g == nil {
		return 0
	}
	return g.capacity
}

func (g *ProcessorWriteGate) InFlight() int {
	if g == nil {
		return 0
	}
	return int(g.inFlight.Load())
}

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
