package monitoring_test

import (
	"os"
	"strings"
	"testing"
)

func TestPrometheusRulesRedisHotPathAlerts(t *testing.T) {
	data, err := os.ReadFile("prometheus.rules.yaml")
	if err != nil {
		t.Fatalf("read prometheus.rules.yaml: %v", err)
	}
	content := string(data)
	for _, want := range []string{
		"alert: RedisLuaLatencyHigh",
		"alert: RedisBreakerOpen",
		"alert: TrackerLatencyP99Critical",
		"alert: TrackerLatencyP99Sustained",
		"alert: ReportQueryLatencyHigh",
		"alert: StreamProducerPostDebitRejected",
		"ad_redis_lua_duration_seconds_bucket",
		"ad_http_request_duration_seconds_bucket",
		"ad_report_query_duration_seconds_bucket",
		"ad_stream_producer_post_debit_rejected_total",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("prometheus.rules.yaml missing %q", want)
		}
	}
	if !strings.Contains(content, "for: 30s") {
		t.Fatal("prometheus.rules.yaml must wire load-test abort at for: 30s on tracker p99")
	}
}
