package compose_test

import (
	"os"
	"strings"
	"testing"
)

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
