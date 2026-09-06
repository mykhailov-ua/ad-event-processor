package payment

import (
	"testing"

	"go.uber.org/goleak"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m,
		goleak.IgnoreTopFunction("ad-event-processor/internal/filter.init.0.func1"),
		goleak.IgnoreTopFunction("ad-event-processor/internal/filter.init.0.func2"),
		goleak.IgnoreTopFunction("ad-event-processor/internal/stream.(*IDRingBuffer).refillWorker"),
		goleak.IgnoreTopFunction("github.com/panjf2000/ants/v2.(*poolCommon).purgeStaleWorkers"),
		goleak.IgnoreTopFunction("github.com/panjf2000/ants/v2.(*poolCommon).ticktock"),
		goleak.IgnoreTopFunction("github.com/jackc/pgx/v5/pgxpool.(*Pool).backgroundHealthCheck"),
		goleak.IgnoreTopFunction("github.com/jackc/pgx/v5/pgxpool.NewWithConfig.func5"),
	)
}
