package opsadmin

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

func ResolveFilterRejectMetricsURL(fallback string) string {
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

func fetchPrometheusMetrics(ctx context.Context, client *http.Client, url string) ([]byte, string, error) {
	if client == nil {
		client = http.DefaultClient
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, "", err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("metrics scrape status %d", resp.StatusCode)
	}
	return body, resp.Header.Get("Content-Type"), nil
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
