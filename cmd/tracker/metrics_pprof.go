package main

import (
	"net/http"
	"net/http/pprof"
	"os"
)

// registerMetricsPprof mounts Go pprof/trace on the metrics sidecar when TRACKER_PPROF_ENABLED=1.
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

// metricsServerWriteTimeout returns HTTP write timeout for the metrics sidecar.
func metricsServerWriteTimeout() int {
	if os.Getenv("TRACKER_PPROF_ENABLED") == "1" {
		return 120
	}
	return 10
}
