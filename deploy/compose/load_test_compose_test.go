package compose_test

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func loadTestEnvFile(t *testing.T) map[string]string {
	t.Helper()
	path := filepath.Join("..", "..", ".env.load-test")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read .env.load-test: %v", err)
	}
	out := make(map[string]string)
	sc := bufio.NewScanner(strings.NewReader(string(data)))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			t.Fatalf("invalid .env.load-test line: %q", line)
		}
		out[key] = val
	}
	if err := sc.Err(); err != nil {
		t.Fatalf("scan .env.load-test: %v", err)
	}
	return out
}

func loadTestInt(t *testing.T, env map[string]string, key string) int {
	t.Helper()
	v, ok := env[key]
	if !ok {
		t.Fatalf(".env.load-test missing %q", key)
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		t.Fatalf(".env.load-test %q=%q: %v", key, v, err)
	}
	return n
}

func loadTestTrackerIngestPort(t *testing.T, env map[string]string, idx int) int {
	t.Helper()
	base := loadTestInt(t, env, "LOAD_TEST_TRACKER_INGEST_BASE")
	step := loadTestInt(t, env, "LOAD_TEST_TRACKER_INGEST_STEP")
	return base + idx*step
}

func loadTestTrackerMetricsPort(t *testing.T, env map[string]string, idx int) int {
	t.Helper()
	base := loadTestInt(t, env, "LOAD_TEST_TRACKER_METRICS_BASE")
	step := loadTestInt(t, env, "LOAD_TEST_TRACKER_METRICS_STEP")
	return base + idx*step
}

func loadTestRedisPort(t *testing.T, env map[string]string, shard int) int {
	t.Helper()
	base := loadTestInt(t, env, "LOAD_TEST_REDIS_BASE")
	step := loadTestInt(t, env, "LOAD_TEST_REDIS_STEP")
	return base + shard*step
}

func TestLoadTestOverlayEnablesLocalQuantaLive(t *testing.T) {
	data, err := os.ReadFile("docker-compose.load-test.yaml")
	if err != nil {
		t.Fatalf("read docker-compose.load-test.yaml: %v", err)
	}
	content := string(data)
	for _, want := range []string{
		"QUOTA_MODE: live",
		"LOCAL_QUOTA_MODE: live",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("docker-compose.load-test.yaml missing %q", want)
		}
	}
}

func TestLoadTestComposeUsesEnvVars(t *testing.T) {
	data, err := os.ReadFile("docker-compose.load-test.yaml")
	if err != nil {
		t.Fatalf("read docker-compose.load-test.yaml: %v", err)
	}
	content := string(data)
	for _, want := range []string{
		"SERVER_PORT: ${LOAD_TEST_TRACKER_0_INGEST_PORT}",
		"METRICS_PORT: ${LOAD_TEST_TRACKER_0_METRICS_PORT}",
		"SERVER_PORT: ${LOAD_TEST_TRACKER_3_INGEST_PORT}",
		"PROCESSOR_PORT=${LOAD_TEST_PROCESSOR_PORT}",
		"REDIS_ADDRS: ${LOAD_TEST_REDIS_ADDRS}",
		"'${LOAD_TEST_REDIS_0_PORT}:6379'",
		"'${LOAD_TEST_REDIS_5_PORT}:6379'",
		"--web.listen-address=:${LOAD_TEST_PROMETHEUS_PORT}",
		"--web.listen-address=:${LOAD_TEST_ALERTMANAGER_PORT}",
		"REDIS_PORT=${LOAD_TEST_REDIS_PORT}",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("docker-compose.load-test.yaml missing %q", want)
		}
	}
}

func TestLoadTestNginxRenderedFromEnvSource(t *testing.T) {
	env := loadTestEnvFile(t)
	count := loadTestInt(t, env, "LOAD_TEST_TRACKER_COUNT")
	edge := loadTestInt(t, env, "LOAD_TEST_EDGE_PORT")
	control := loadTestInt(t, env, "LOAD_TEST_CONTROL_PORT")

	data, err := os.ReadFile("../../deploy/nginx/nginx.load-test.conf")
	if err != nil {
		t.Fatalf("read nginx.load-test.conf: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, fmt.Sprintf("listen %d", edge)) {
		t.Fatalf("nginx.load-test.conf missing edge listen %d", edge)
	}
	if !strings.Contains(content, fmt.Sprintf("127.0.0.1:%d", control)) {
		t.Fatalf("nginx.load-test.conf missing control upstream %d", control)
	}
	first := loadTestTrackerIngestPort(t, env, 0)
	last := loadTestTrackerIngestPort(t, env, count-1)
	if !strings.Contains(content, fmt.Sprintf("127.0.0.1:%d", first)) {
		t.Fatalf("nginx.load-test.conf missing tracker-0 %d", first)
	}
	if !strings.Contains(content, fmt.Sprintf("127.0.0.1:%d", last)) {
		t.Fatalf("nginx.load-test.conf missing tracker-%d %d", count-1, last)
	}
}

func TestLoadTestPrometheusRenderedFromEnvSource(t *testing.T) {
	env := loadTestEnvFile(t)
	constrained := loadTestInt(t, env, "LOAD_TEST_CONSTRAINED_TRACKERS")
	alertmanager := loadTestInt(t, env, "LOAD_TEST_ALERTMANAGER_PORT")
	edge := loadTestInt(t, env, "LOAD_TEST_EDGE_PORT")
	processor := loadTestInt(t, env, "LOAD_TEST_PROCESSOR_PORT")

	data, err := os.ReadFile("../../deploy/monitoring/prometheus.load-test.yaml")
	if err != nil {
		t.Fatalf("read prometheus.load-test.yaml: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, fmt.Sprintf("127.0.0.1:%d", alertmanager)) {
		t.Fatalf("prometheus.load-test.yaml missing alertmanager %d", alertmanager)
	}
	if !strings.Contains(content, fmt.Sprintf("127.0.0.1:%d", edge)) {
		t.Fatalf("prometheus.load-test.yaml missing edge %d", edge)
	}
	if !strings.Contains(content, fmt.Sprintf("127.0.0.1:%d", processor)) {
		t.Fatalf("prometheus.load-test.yaml missing processor %d", processor)
	}
	for i := 0; i < constrained; i++ {
		metrics := loadTestTrackerMetricsPort(t, env, i)
		want := fmt.Sprintf("127.0.0.1:%d", metrics)
		if !strings.Contains(content, want) {
			t.Fatalf("prometheus.load-test.yaml missing %q", want)
		}
	}
}
