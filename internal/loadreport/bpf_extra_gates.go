package loadreport

import (
	"strconv"
	"strings"
)

func checkBPFExtraGatesPrometheus(prom *promClient, rateWindow string) []BPFGateCheck {
	rate := func(q string) string {
		return strings.ReplaceAll(q, "${window}", rateWindow)
	}

	var checks []BPFGateCheck

	pgAcquire := prom.scalar(`histogram_quantile(0.99, sum(rate(ad_processor_pg_acquire_wait_seconds_bucket[${window}])) by (le)) * 1000`)
	pgVal, pgOk := parseFloatOrNa(pgAcquire)
	checks = append(checks, BPFGateCheck{
		Name:   "processor_pg_acquire_wait_p99_ms",
		Value:  pgAcquire,
		Limit:  "100",
		OK:     !pgOk || pgVal < 100,
		Detail: "processor PG connection pool acquisition wait p99",
	})

	gcPause := prom.scalar(`histogram_quantile(0.99, sum(rate(go_gc_duration_seconds_bucket[${window}])) by (le)) * 1000`)
	gcVal, gcOk := parseFloatOrNa(gcPause)
	checks = append(checks, BPFGateCheck{
		Name:   "go_gc_pause_p99_ms",
		Value:  gcPause,
		Limit:  "5",
		OK:     !gcOk || gcVal < 5,
		Detail: "Go runtime GC stop-the-world pause p99",
	})

	keygenDepth := prom.scalarOrZero("ad_region_proxy_keygen_queue_depth")
	depthVal, _ := strconv.ParseFloat(keygenDepth, 64)
	checks = append(checks, BPFGateCheck{
		Name:   "region_proxy_keygen_queue_depth",
		Value:  keygenDepth,
		Limit:  "100",
		OK:     depthVal <= 100,
		Detail: "region-proxy WAL keygen queue backlog",
	})

	keygenLag := prom.scalar(`histogram_quantile(0.99, sum(rate(ad_region_proxy_keygen_lag_seconds_bucket[${window}])) by (le)) * 1000`)
	lagVal, lagOk := parseFloatOrNa(keygenLag)
	checks = append(checks, BPFGateCheck{
		Name:   "region_proxy_keygen_lag_p99_ms",
		Value:  keygenLag,
		Limit:  "1000",
		OK:     !lagOk || lagVal < 1000,
		Detail: "region-proxy WAL keygen latency p99",
	})

	streamDepth := prom.scalar(`max(quantile(0.99, ad_stream_producer_queue_depth))`)
	streamVal, streamOk := parseFloatOrNa(streamDepth)
	checks = append(checks, BPFGateCheck{
		Name:   "stream_producer_queue_depth_p99",
		Value:  streamDepth,
		Limit:  "1000",
		OK:     !streamOk || streamVal < 1000,
		Detail: "stream producer async queue backlog (hot-path goroutine pressure)",
	})

	localQuotaBlock := prom.scalarOrZero(rate(`sum(rate(ad_tracker_local_quota_block_total[${window}]))`))
	quotaVal, _ := strconv.ParseFloat(localQuotaBlock, 64)
	checks = append(checks, BPFGateCheck{
		Name:   "local_quota_block_rate",
		Value:  localQuotaBlock,
		Limit:  "100",
		OK:     quotaVal <= 100,
		Detail: "local quota block events per second (ledger pressure)",
	})

	xdpPinned := prom.scalar("ad_xdp_pinned_map_count")
	if xdpPinned != "na" && xdpPinned != "" {
		xdpVal, _ := strconv.ParseFloat(xdpPinned, 64)
		checks = append(checks, BPFGateCheck{
			Name:   "xdp_pinned_map_count",
			Value:  xdpPinned,
			Limit:  "3",
			OK:     xdpVal >= 3,
			Detail: "XDP pinned BPF map count (expected stable when edge-xdp is enabled)",
		})
	}

	return checks
}

func parseFloatOrNa(s string) (float64, bool) {
	if s == "na" || s == "" {
		return 0, false
	}
	v, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	return v, err == nil
}
