package monitoring_test

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestMainGrafanaDashboardHotPathPanels(t *testing.T) {
	data, err := os.ReadFile("grafana-provisioning/dashboards/main.json")
	if err != nil {
		t.Fatalf("read main.json: %v", err)
	}
	var dash map[string]interface{}
	if err := json.Unmarshal(data, &dash); err != nil {
		t.Fatalf("parse main.json: %v", err)
	}
	panels, ok := dash["panels"].([]interface{})
	if !ok || len(panels) == 0 {
		t.Fatal("main.json has no panels")
	}
	joined := string(data)
	for _, want := range []string{
		"ad_redis_lua_duration_seconds_bucket",
		"ad_redis_breaker_state",
		"ad_http_request_duration_seconds_bucket",
		"ad_stream_producer_post_debit_rejected_total",
		"ad_stream_producer_queue_depth",
		"ad_redis_ops_total",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("main.json missing panel expr %q", want)
		}
	}
}
