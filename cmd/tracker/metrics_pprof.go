// Metrics sidecar helpers: optional pprof on METRICS_PORT (not the gnet ingest listener).
//
// Boundary: ingest stays on gnet (SERVER_PORT or TRACKER_UNIX_SOCKET). METRICS_PORT serves
// /metrics, /health, /ready, and optional /debug/pprof/* only; no /track traffic.
package main

import (
	"net/http"
	"net/http/pprof"
	"os"
)

// registerMetricsPprof mounts /debug/pprof/* when TRACKER_PPROF_ENABLED=1.
// Cold-path only; never on SERVER_PORT ingest listener. Profiling Tier B pinned workers
// is safe because pprof samples all goroutines; ingest latency is unaffected by sidecar bind.
func registerMetricsPprof(mux *http.ServeMux) {
	if os.Getenv("TRACKER_PPROF_ENABLED") != "1" {
		return
	}
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
}

// metricsServerWriteTimeout returns HTTP WriteTimeout seconds for the metrics sidecar.
// 120s when TRACKER_PPROF_ENABLED=1 (long /debug/pprof/profile captures); 10s otherwise.
// Does not affect gnet ingest deadlines (FILTER_TIMEOUT_MS, WRITE_TIMEOUT_MS).
func metricsServerWriteTimeout() int {
	if os.Getenv("TRACKER_PPROF_ENABLED") == "1" {
		return 120
	}
	return 10
}
