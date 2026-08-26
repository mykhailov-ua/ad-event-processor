package compose_test

import (
	"os"
	"strings"
	"testing"
)

func TestDockerfileKeepsSymbolsForBPFTraceTracker(t *testing.T) {
	data, err := os.ReadFile("../docker/Dockerfile")
	if err != nil {
		t.Fatalf("read Dockerfile: %v", err)
	}
	content := string(data)
	for _, want := range []string{
		`TRACKER_BPF_TRACE" = "1"`,
		`-ldflags="-w" -o /bin/tracker`,
		`-ldflags="-s -w" -o /bin/tracker`,
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("Dockerfile missing %q", want)
		}
	}
}
