package main

import (
	"net/http"
	"net/http/pprof"
	"os"
)

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

func metricsServerWriteTimeout() int {
	if os.Getenv("TRACKER_PPROF_ENABLED") == "1" {
		return 120
	}
	return 10
}
