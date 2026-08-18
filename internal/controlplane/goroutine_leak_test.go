package controlplane

import (
	"testing"

	"go.uber.org/goleak"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m,
		goleak.IgnoreTopFunction("github.com/bidshard/ad-event-processor/internal/ingestion.init.0.func1"),
		goleak.IgnoreTopFunction("github.com/bidshard/ad-event-processor/internal/ingestion.init.0.func2"),
		goleak.IgnoreTopFunction("github.com/bidshard/ad-event-processor/internal/ingestion.(*IDRingBuffer).refillWorker"),
		goleak.IgnoreTopFunction("github.com/panjf2000/ants/v2.(*poolCommon).purgeStaleWorkers"),
		goleak.IgnoreTopFunction("github.com/panjf2000/ants/v2.(*poolCommon).ticktock"),
	)
}
