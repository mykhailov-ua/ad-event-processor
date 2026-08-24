package ingestion

import (
	"testing"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/domain"
	"github.com/google/uuid"
	redis "github.com/redis/go-redis/v9"
)

func benchStreamProducer() *StreamProducer {
	rdb := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1"})
	return NewStreamProducer(rdb, "bench-stream", 1000, 0)
}

func BenchmarkStreamProducer_Process(b *testing.B) {
	p := benchStreamProducer()
	defer p.Close()

	evt := &domain.Event{
		ClickID:    "clk-bench",
		CampaignID: uuid.MustParse("33333333-3333-3333-3333-333333333333"),
		Type:       "click",
		Payload:    []byte(`{"k":"v"}`),
	}

	b.ReportAllocs()
	for b.Loop() {
		_ = p.Process(evt)
	}
}

func BenchmarkStreamProducer_AdmissionCheck(b *testing.B) {
	p := benchStreamProducer()
	defer p.Close()

	cfg := &config.Config{StreamProducerAdmissionPct: 85}
	sharder := NewJumpHashSharder(1)
	producers := []*StreamProducer{p}
	campaignID := uuid.MustParse("44444444-4444-4444-4444-444444444444")

	b.ReportAllocs()
	for b.Loop() {
		_, _, _ = tryAcquireStreamAdmission(cfg, sharder, producers, nil, campaignID)
	}
}
