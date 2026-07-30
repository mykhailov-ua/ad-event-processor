package compose_test

import (
	"os"
	"strings"
	"testing"
)

func TestProductionOverlayDocumentsTLS(t *testing.T) {
	data, err := os.ReadFile("production.yaml")
	if err != nil {
		t.Fatalf("read production.yaml: %v", err)
	}
	content := string(data)
	for _, want := range []string{
		"ssl=on",
		"verify-full",
		"tls-port",
		"EVENTS_RETENTION_DAYS",
		"EVENTS_HASH_IP_AT_INSERT",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("production.yaml missing %q", want)
		}
	}
}
