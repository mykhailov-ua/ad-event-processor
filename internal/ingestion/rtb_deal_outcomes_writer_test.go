package ingestion

import (
	"testing"

	"github.com/bidshard/ad-event-processor/internal/rtb"
)

func TestRecordRtbDealOutcome_noWriter(t *testing.T) {
	globalRtbOutcomeWriter.Store(nil)
	recordRtbDealOutcome("deal-a", 100, rtb.AuctionResult{}, rtb.NoBidNone)
}

func TestRtbDealOutcomeWriter_enqueue(t *testing.T) {
	w := &RtbDealOutcomeWriter{}
	if !w.Enqueue([]byte("deal-a"), 1, 100) {
		t.Fatal("expected enqueue to succeed on uninitialized writer")
	}
}
