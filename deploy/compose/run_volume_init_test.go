package compose_test

import (
	"os"
	"strings"
	"testing"
)

func TestDockerComposeRunDirInit(t *testing.T) {
	data, err := os.ReadFile("docker-compose.yaml")
	if err != nil {
		t.Fatalf("read docker-compose.yaml: %v", err)
	}
	content := string(data)
	for _, want := range []string{
		"run-dir-init:",
		"init-run-volume.sh",
		"/run/ad-event-processor/redis",
		"ad_event_processor_logs:/var/log/ad-event-processor",
		"ad_event_processor_ch_spool:/var/spool/ad-event-processor/ch",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("docker-compose.yaml missing %q", want)
		}
	}
}

func TestInitRunVolumeScriptCreatesRedisDir(t *testing.T) {
	data, err := os.ReadFile("scripts/init-run-volume.sh")
	if err != nil {
		t.Fatalf("read init-run-volume.sh: %v", err)
	}
	content := string(data)
	for _, want := range []string{
		"redis",
		"/var/log/ad-event-processor",
		"/var/spool/ad-event-processor/ch",
		"offsets",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("init-run-volume.sh missing %q", want)
		}
	}
}

func TestLoadTestComposeTrackerLabTransport(t *testing.T) {
	data, err := os.ReadFile("docker-compose.load-test.yaml")
	if err != nil {
		t.Fatalf("read docker-compose.load-test.yaml: %v", err)
	}
	content := string(data)
	for _, want := range []string{
		"TRACKER_UNIX_SOCKET: ''",
		"TRANSPORT_USE_UDS: '0'",
		"HTTP1_BODY_IDLE_MS: '60000'",
		"TRACKER_BPF_TRACE: '1'",
		"'--appendonly'",
		"'no'",
		"cpus: '0.75'",
		"--health-probe', '${LOAD_TEST_TRACKER_0_HEALTH_URL}'",
		"x-tracker-healthcheck:",
		"'--health-probe', 'http://127.0.0.1:${LOAD_TEST_PROCESSOR_PORT}/health'",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("docker-compose.load-test.yaml missing %q", want)
		}
	}
}
