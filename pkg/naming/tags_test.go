package naming_test

import (
	"testing"

	"github.com/bidshard/ad-event-processor/pkg/naming"
)

func TestBPFTraceBuildTag(t *testing.T) {
	if got := naming.BPFTraceBuildTag(); got != "ad_event_processor_bpf_trace" {
		t.Fatalf("BPFTraceBuildTag() = %q, want ad_event_processor_bpf_trace", got)
	}
}
