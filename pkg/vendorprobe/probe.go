// Package vendorprobe runs optional third-party dependency health probes.
package vendorprobe

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

type Probe interface {
	Name() string
	Probe(ctx context.Context) error
}

type Observer interface {
	ObserveProbe(vendor string, success bool, latency time.Duration)
	ObserveProbeError(vendor string)
}

type Registry struct {
	probes []Probe
}

func NewRegistry() *Registry {
	return &Registry{}
}

func (r *Registry) Register(p Probe) {
	if r == nil || p == nil {
		return
	}
	r.probes = append(r.probes, p)
}

func (r *Registry) Probes() []Probe {
	if r == nil {
		return nil
	}
	out := make([]Probe, len(r.probes))
	copy(out, r.probes)
	return out
}

type WorkerConfig struct {
	Interval time.Duration
	Timeout  time.Duration
}

type Worker struct {
	reg      *Registry
	cfg      WorkerConfig
	observer Observer
}

func NewWorker(reg *Registry, cfg WorkerConfig, observer Observer) *Worker {
	if cfg.Interval <= 0 {
		cfg.Interval = 60 * time.Second
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 5 * time.Second
	}
	return &Worker{reg: reg, cfg: cfg, observer: observer}
}

func (w *Worker) Start(ctx context.Context) {
	if w == nil || w.reg == nil || len(w.reg.probes) == 0 {
		return
	}
	ticker := time.NewTicker(w.cfg.Interval)
	defer ticker.Stop()

	w.runOnce(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.runOnce(ctx)
		}
	}
}

func (w *Worker) runOnce(parent context.Context) {
	probes := w.reg.Probes()
	var wg sync.WaitGroup
	for _, p := range probes {
		probe := p
		wg.Add(1)
		go func() {
			defer wg.Done()
			w.runProbe(parent, probe)
		}()
	}
	wg.Wait()
}

func (w *Worker) runProbe(parent context.Context, p Probe) {
	ctx, cancel := context.WithTimeout(parent, w.cfg.Timeout)
	defer cancel()

	start := time.Now()
	err := p.Probe(ctx)
	latency := time.Since(start)
	success := err == nil

	if err != nil {
		slog.Warn("vendor probe failed", "vendor", p.Name(), "error", err, "latency_ms", latency.Milliseconds())
		if w.observer != nil {
			w.observer.ObserveProbeError(p.Name())
		}
	}
	if w.observer != nil {
		w.observer.ObserveProbe(p.Name(), success, latency)
	}
}
