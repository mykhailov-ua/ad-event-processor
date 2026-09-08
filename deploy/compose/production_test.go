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
		"FILTER_TIMEOUT_MS=100",
		"STREAM_PRODUCER_ADMISSION_PCT=85",
		"LOCAL_QUOTA_MODE=live",
		"QUOTA_MODE=live",
		"TRACKER_PG_FALLBACK=0",
		"TRANSPORT_USE_UDS=1",
		"PROCESSOR_STREAM_LAG_MAX_SEC",
		"CH_SPOOL_DIR=/var/spool/ad-event-processor/ch",
		"CH_INGEST_SOURCE=broker",
		"BROKER_SHADOW_MODE",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("production.yaml missing %q", want)
		}
	}
}
