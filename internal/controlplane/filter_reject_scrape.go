package controlplane

import (
	"os"
	"strings"
)

func resolveFilterRejectMetricsURL(fallback string) string {
	if v := strings.TrimSpace(os.Getenv("FILTER_REJECT_METRICS_SCRAPE_URL")); v != "" {
		return v
	}
	if v := strings.TrimSpace(os.Getenv("TRACKER_METRICS_SCRAPE_URL")); v != "" {
		return v
	}
	if v := strings.TrimSpace(fallback); v != "" {
		return v
	}
	return "http://127.0.0.1:9101/metrics"
}

func edgeBlockedToRejectCounters(blocked map[string]uint64) map[string]float64 {
	if len(blocked) == 0 {
		return nil
	}
	out := make(map[string]float64, len(blocked))
	for kind, val := range blocked {
		kind = strings.TrimSpace(kind)
		if kind == "" || val == 0 {
			continue
		}
		out["edge_"+kind] = float64(val)
	}
	return out
}

func mergeRejectCounterMaps(parts ...map[string]float64) map[string]float64 {
	out := make(map[string]float64)
	for _, part := range parts {
		for kind, val := range part {
			out[kind] += val
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func filterRejectRollupDeltas(previous, current map[string]float64) map[string]uint64 {
	out := make(map[string]uint64)
	for kind, cur := range current {
		delta := filterRejectCounterDelta(previous[kind], cur)
		if delta > 0 {
			out[kind] = delta
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
