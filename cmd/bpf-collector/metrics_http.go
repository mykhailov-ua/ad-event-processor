package main

import (
	"context"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"ad-event-processor/pkg/lifecycle"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const bpfMetricPrefix = "ad_event_processor_bpf_"

type promExporter struct {
	run *probeRun

	oncpuPct        *prometheus.GaugeVec
	runqueueP99Us   *prometheus.GaugeVec
	connectAvgUs    *prometheus.GaugeVec
	tcpRetrans      *prometheus.GaugeVec
	ctxSwitchPerSec *prometheus.GaugeVec

	mu sync.Mutex
}

func newPromExporter(run *probeRun) *promExporter {
	e := &promExporter{run: run}
	e.oncpuPct = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: bpfMetricPrefix + "oncpu_pct",
		Help: "BPF measured on-CPU percent by role during session wall time",
	}, []string{"role", "name"})
	e.runqueueP99Us = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: bpfMetricPrefix + "runqueue_p99_us",
		Help: "Runqueue wait p99 microseconds by role",
	}, []string{"role", "name"})
	e.connectAvgUs = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: bpfMetricPrefix + "connect_avg_us",
		Help: "Average connect syscall duration microseconds",
	}, []string{"role", "name"})
	e.tcpRetrans = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: bpfMetricPrefix + "tcp_retrans_total",
		Help: "TCP retransmit count observed during session",
	}, []string{"role", "name"})
	e.ctxSwitchPerSec = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: bpfMetricPrefix + "ctx_switch_per_sec",
		Help: "Context switches per second by role",
	}, []string{"role", "name"})
	prometheus.MustRegister(e.oncpuPct, e.runqueueP99Us, e.connectAvgUs, e.tcpRetrans, e.ctxSwitchPerSec)
	return e
}

func (e *promExporter) refresh() {
	duration := sessionDuration(e.run.session)
	pidStats, err := e.run.aggregatePIDStats(duration)
	if err != nil {
		slog.Debug("prom refresh pid stats", "error", err)
		return
	}
	netStats, err := e.run.aggregateNet()
	if err != nil {
		slog.Debug("prom refresh net stats", "error", err)
	}

	e.mu.Lock()
	defer e.mu.Unlock()
	e.oncpuPct.Reset()
	e.runqueueP99Us.Reset()
	e.ctxSwitchPerSec.Reset()
	e.connectAvgUs.Reset()
	e.tcpRetrans.Reset()

	for i := range pidStats {
		s := &pidStats[i]
		lbl := prometheus.Labels{"role": s.Role, "name": s.Name}
		e.oncpuPct.With(lbl).Set(s.OnCPUPct)
		e.runqueueP99Us.With(lbl).Set(s.RunqueueP99Us)
		e.ctxSwitchPerSec.With(lbl).Set(s.CtxSwitchPerSec)
	}
	for _, row := range netStats {
		role, _ := row["role"].(string)
		name, _ := row["name"].(string)
		lbl := prometheus.Labels{"role": role, "name": name}
		if v, ok := row["connect_avg_us"].(float64); ok {
			e.connectAvgUs.With(lbl).Set(v)
		}
		switch v := row["retrans"].(type) {
		case uint64:
			e.tcpRetrans.With(lbl).Set(float64(v))
		case int:
			e.tcpRetrans.With(lbl).Set(float64(v))
		}
	}
}

func (r *probeRun) serveMetrics(ctx context.Context, addr string) {
	exporter := newPromExporter(r)
	mux := http.NewServeMux()
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, req *http.Request) {
		exporter.refresh()
		promhttp.Handler().ServeHTTP(w, req)
	})
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	srv := &http.Server{Addr: addr, Handler: mux}
	lifecycle.ApplySidecarHTTPServerTimeouts(srv)
	go func() {
		<-ctx.Done()
		_ = srv.Close()
	}()
	slog.Info("bpf metrics listening", "addr", addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Warn("metrics server", "error", err)
	}
}

func (r *probeRun) dumpLoop(ctx context.Context) {
	ticker := time.NewTicker(r.dumpInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := r.dumpMaps(); err != nil {
				slog.Warn("periodic dump", "error", err)
			}
		}
	}
}
