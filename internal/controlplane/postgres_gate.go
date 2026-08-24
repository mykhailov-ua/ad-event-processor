package controlplane

import (
	"context"
	"errors"
	"sync/atomic"
	"time"

	"ad-event-processor/internal/metrics"
)

var ErrPostgresGateRejected = errors.New("postgres gate rejected")

const postgresGateReserve = 1

type PostgresGate struct {
	sem      chan struct{}
	capacity int
	lowSlots chan struct{}
	inFlight atomic.Int32
}

func NewPostgresGate(maxConns int) *PostgresGate {
	capacity := maxConns - postgresGateReserve
	if capacity < 2 {
		capacity = 2
	}
	lowCap := capacity - 1
	if lowCap < 1 {
		lowCap = 1
	}
	return &PostgresGate{
		sem:      make(chan struct{}, capacity),
		capacity: capacity,
		lowSlots: make(chan struct{}, lowCap),
	}
}

func (g *PostgresGate) AcquireHigh(ctx context.Context) error {
	if g == nil {
		return nil
	}
	start := time.Now()
	select {
	case g.sem <- struct{}{}:
		if wait := time.Since(start); wait > 0 {
			metrics.PostgresGateAcquireWaitSeconds.WithLabelValues("high").Observe(wait.Seconds())
		}
		g.inFlight.Add(1)
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (g *PostgresGate) ReleaseHigh() {
	if g == nil {
		return
	}
	g.inFlight.Add(-1)
	<-g.sem
}

func (g *PostgresGate) AcquireLow(ctx context.Context) error {
	if g == nil {
		return nil
	}
	select {
	case g.lowSlots <- struct{}{}:
	case <-ctx.Done():
		return ctx.Err()
	default:
		metrics.PostgresGateRejectedTotal.WithLabelValues("low").Inc()
		return ErrPostgresGateRejected
	}

	start := time.Now()
	select {
	case g.sem <- struct{}{}:
		if wait := time.Since(start); wait > 0 {
			metrics.PostgresGateAcquireWaitSeconds.WithLabelValues("low").Observe(wait.Seconds())
		}
		g.inFlight.Add(1)
		return nil
	case <-ctx.Done():
		<-g.lowSlots
		return ctx.Err()
	}
}

func (g *PostgresGate) ReleaseLow() {
	if g == nil {
		return
	}
	g.inFlight.Add(-1)
	<-g.sem
	<-g.lowSlots
}

func (g *PostgresGate) InFlight() int {
	if g == nil {
		return 0
	}
	return int(g.inFlight.Load())
}

func (s *Service) withPgHigh(ctx context.Context, fn func(context.Context) error) error {
	if s == nil || s.pgGate == nil {
		return fn(ctx)
	}
	if err := s.pgGate.AcquireHigh(ctx); err != nil {
		return err
	}
	defer s.pgGate.ReleaseHigh()
	return fn(ctx)
}

func (s *Service) withPgLow(ctx context.Context, fn func(context.Context) error) error {
	if s == nil || s.pgGate == nil {
		return fn(ctx)
	}
	if err := s.pgGate.AcquireLow(ctx); err != nil {
		return err
	}
	defer s.pgGate.ReleaseLow()
	return fn(ctx)
}
