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
		"'--health-probe', 'http://127.0.0.1:${LOAD_TEST_PROCESSOR_PORT}/health'",
		"'--health-probe', 'http://127.0.0.1:${LOAD_TEST_CONTROL_PORT}/healthz'",
		"CONTROL_ENABLE_MARGIN_GUARD=0",
		"CONTROL_ENABLE_COST_SYNC=0",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("docker-compose.load-test.yaml missing %q", want)
		}
	}
}

func TestLoadTestNginxTemplateWiresEnvVars(t *testing.T) {
	env := loadTestEnvFile(t)
	count := loadTestInt(t, env, "LOAD_TEST_TRACKER_COUNT")

	data, err := os.ReadFile("../../deploy/nginx/nginx.load-test.conf.in")
	if err != nil {
		t.Fatalf("read nginx.load-test.conf.in: %v", err)
	}
	content := string(data)
	for _, want := range []string{
		"listen ${LOAD_TEST_EDGE_PORT}",
		"server 127.0.0.1:${LOAD_TEST_CONTROL_PORT}",
		"server 127.0.0.1:${LOAD_TEST_TRACKER_0_INGEST_PORT}",
		fmt.Sprintf("server 127.0.0.1:${LOAD_TEST_TRACKER_%d_INGEST_PORT}", count-1),
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("nginx.load-test.conf.in missing %q", want)
		}
	}
}

func TestLoadTestPrometheusTemplateWiresEnvVars(t *testing.T) {
	env := loadTestEnvFile(t)
	constrained := loadTestInt(t, env, "LOAD_TEST_CONSTRAINED_TRACKERS")

	data, err := os.ReadFile("../../deploy/monitoring/prometheus.load-test.yaml.in")
	if err != nil {
		t.Fatalf("read prometheus.load-test.yaml.in: %v", err)
	}
	content := string(data)
	for _, want := range []string{
		"127.0.0.1:${LOAD_TEST_ALERTMANAGER_PORT}",
		"127.0.0.1:${LOAD_TEST_CONTROL_PORT}",
		"127.0.0.1:${LOAD_TEST_EDGE_PORT}",
		"127.0.0.1:${LOAD_TEST_PROCESSOR_PORT}",
		"127.0.0.1:${LOAD_TEST_TRACKER_0_METRICS_PORT}",
		fmt.Sprintf("127.0.0.1:${LOAD_TEST_TRACKER_%d_METRICS_PORT}", constrained-1),
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("prometheus.load-test.yaml.in missing %q", want)
		}
	}
}
