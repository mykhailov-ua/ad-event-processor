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
	if !strings.Contains(string(data), "redis") {
		t.Fatal("init-run-volume.sh must mkdir redis socket parent")
	}
}

func TestLoadTestComposeTrackerTCPHealth(t *testing.T) {
	data, err := os.ReadFile("docker-compose.load-test.yaml")
	if err != nil {
		t.Fatalf("read docker-compose.load-test.yaml: %v", err)
	}
	content := string(data)
	for _, want := range []string{
		"TRACKER_UNIX_SOCKET: ''",
		"TRANSPORT_USE_UDS: '0'",
		"TRACKER_BPF_TRACE: '1'",
		"--health-probe', '${LOAD_TEST_TRACKER_0_HEALTH_URL}'",
		"x-tracker-healthcheck:",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("docker-compose.load-test.yaml missing %q", want)
		}
	}
}
