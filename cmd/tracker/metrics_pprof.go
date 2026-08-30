// Metrics sidecar helpers: optional pprof on METRICS_PORT (not the gnet ingest listener).
package main

import (
	"net/http"
	"net/http/pprof"
	"os"
)

// registerMetricsPprof mounts /debug/pprof/* when TRACKER_PPROF_ENABLED=1.
// Cold-path only; never on SERVER_PORT ingest listener.
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
// 120s when pprof enabled (long profile captures); 10s otherwise.
func metricsServerWriteTimeout() int {
	if os.Getenv("TRACKER_PPROF_ENABLED") == "1" {
		return 120
	}
	return 10
}
