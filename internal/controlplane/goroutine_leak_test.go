package controlplane

import (
	"testing"

	"go.uber.org/goleak"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m,
		goleak.IgnoreTopFunction("ad-event-processor/internal/ingest.init.0.func1"),
		goleak.IgnoreTopFunction("ad-event-processor/internal/ingest.init.0.func2"),
		goleak.IgnoreTopFunction("ad-event-processor/internal/filter.init.0.func1"),
		goleak.IgnoreTopFunction("ad-event-processor/internal/filter.init.0.func2"),
		goleak.IgnoreTopFunction("github.com/ClickHouse/clickhouse-go/v2.(*connPool).runDrainPool"),
		goleak.IgnoreTopFunction("ad-event-processor/internal/stream.(*IDRingBuffer).refillWorker"),
		goleak.IgnoreTopFunction("github.com/panjf2000/ants/v2.(*poolCommon).purgeStaleWorkers"),
		goleak.IgnoreTopFunction("github.com/panjf2000/ants/v2.(*poolCommon).ticktock"),
		goleak.IgnoreTopFunction("github.com/redis/go-redis/v9/maintnotifications.(*CircuitBreakerManager).cleanupLoop"),
	)
}
