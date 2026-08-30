//go:build !race

package ingest

import (
	"context"
	"strconv"
	"testing"
	"time"

	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
	redis "github.com/redis/go-redis/v9"
)

type benchNoopRedis struct{ mockRedisClient }

func BenchmarkAcceptLocalQuantaFullSkip(b *testing.B) {
	ledger := NewLocalQuantaLedger()
	idem := NewLocalClickIdemCache(time.Hour)
	shards := []redis.UniversalClient{&benchNoopRedis{}}
	stream := NewLocalQuantaStreamPublisherForTest(shards, "events", 1000, idem, time.Millisecond)

	campID := uuid.New()
	custID := uuid.New()
	ledger.Credit(campID, int64(b.N)*10_000+1_000_000, testQuotaChunkMicro)

	f := NewUnifiedFilter(
		shards,
		NewJumpHashSharder(1),
		&mockRegistry{},
		nil,
		0,
		time.Minute,
		time.Hour,
		time.Hour,
		100_000,
		10_000,
		"events",
		1000,
	)
	f.SetQuotaConfig("live", testQuotaChunkMicro, testQuotaRefillThreshold)
	f.SetLuaFastPathEnabled(true)
	f.SetLocalQuantaDeps(LocalQuantaDeps{Ledger: ledger, Stream: stream, Idem: stream.IdemCache()})
	f.SetLocalQuantaMode("live")

	camp := &domain.Campaign{
		ID:         campID,
		CustomerID: custID,
		PacingMode: domain.PacingModeAsap,
	}
	enrichMockCampaign(camp)

	evt := &domain.Event{
		Type:       "impression",
		CampaignID: campID,
		UserID:     "bench-accept",
		IP:         "203.0.113.89",
	}
	clickScratch := evt.ClickIDBuf[:0]
	const amount = int64(10_000)

	b.ReportAllocs()
	benchN := 0
	for b.Loop() {
		buf := strconv.AppendInt(clickScratch[:0], int64(benchN), 10)
		copy(evt.ClickIDBuf[:], buf)
		evt.ClickID = unsafeString(evt.ClickIDBuf[:len(buf)])
		_ = f.AcceptLocalQuantaFullSkip(context.Background(), evt, camp, amount, 0)
		benchN++
	}
}

func BenchmarkLocalQuanta_FullSkip(b *testing.B) {
	ledger := NewLocalQuantaLedger()
	idem := NewLocalClickIdemCache(time.Hour)
	shards := []redis.UniversalClient{&benchNoopRedis{}}
	stream := NewLocalQuantaStreamPublisherForTest(shards, "events", 1000, idem, time.Millisecond)

	campID := uuid.New()
	custID := uuid.New()
	camp := &domain.Campaign{
		ID:         campID,
		CustomerID: custID,
		PacingMode: domain.PacingModeAsap,
	}
	enrichMockCampaign(camp)
	reg := benchRegistryForCampaign(camp)

	f := NewUnifiedFilter(
		shards,
		NewJumpHashSharder(1),
		reg,
		nil,
		0,
		time.Minute,
		time.Hour,
		time.Hour,
		100_000,
		10_000,
		"events",
		1000,
	)
	f.SetQuotaConfig("live", testQuotaChunkMicro, testQuotaRefillThreshold)
	f.SetLuaFastPathEnabled(true)
	f.SetTTCMin(0)
	f.SetLocalQuantaDeps(LocalQuantaDeps{Ledger: ledger, Stream: stream, Idem: stream.IdemCache()})
	f.SetLocalQuantaMode("live")

	const amount = int64(10_000)
	ctx := context.Background()
	evt := &domain.Event{
		Type:            "click",
		IP:              "203.0.113.70",
		UserID:          "bench-full-skip",
		CampaignID:      campID,
		FilterWorkerIdx: 0,
	}
	clickScratch := evt.ClickIDBuf[:0]
	for i := range 100 {
		buf := strconv.AppendInt(clickScratch[:0], int64(i), 10)
		copy(evt.ClickIDBuf[:], buf)
		evt.ClickID = unsafeString(evt.ClickIDBuf[:len(buf)])
		ledger.Credit(campID, amount, testQuotaChunkMicro)
		_ = f.Check(ctx, evt)
		stream.DrainBench()
		idem.Release(evt.ClickID)
	}
	b.ReportAllocs()
	benchN := 0
	for b.Loop() {
		buf := strconv.AppendInt(clickScratch[:0], int64(benchN+1000), 10)
		if len(buf) > len(evt.ClickIDBuf) {
			b.Fatal("click id overflow")
		}
		copy(evt.ClickIDBuf[:], buf)
		evt.ClickID = unsafeString(evt.ClickIDBuf[:len(buf)])
		ledger.Credit(campID, amount, testQuotaChunkMicro)
		_ = f.Check(ctx, evt)
		stream.DrainBench()
		idem.Release(evt.ClickID)
		benchN++
	}
}

func TestUnifiedFilter_Check_zeroAlloc_localQuantaFullSkip(t *testing.T) {
	ledger := NewLocalQuantaLedger()
	idem := NewLocalClickIdemCache(time.Hour)
	shards := []redis.UniversalClient{&benchNoopRedis{}}
	stream := NewLocalQuantaStreamPublisherForTest(shards, "events", 1000, idem, time.Millisecond)

	campID := uuid.New()
	custID := uuid.New()
	ledger.Credit(campID, 50_000_000, testQuotaChunkMicro)

	camp := &domain.Campaign{
		ID:         campID,
		CustomerID: custID,
		PacingMode: domain.PacingModeAsap,
	}
	enrichMockCampaign(camp)
	reg := benchRegistryForCampaign(camp)

	f := NewUnifiedFilter(
		shards,
		NewJumpHashSharder(1),
		reg,
		nil,
		0,
		time.Minute,
		time.Hour,
		time.Hour,
		100_000,
		10_000,
		"events",
		1000,
	)
	f.SetQuotaConfig("live", testQuotaChunkMicro, testQuotaRefillThreshold)
	f.SetLuaFastPathEnabled(true)
	f.SetLocalQuantaDeps(LocalQuantaDeps{Ledger: ledger, Stream: stream, Idem: stream.IdemCache()})
	f.SetLocalQuantaMode("live")

	evt := &domain.Event{
		Type:            "click",
		CampaignID:      campID,
		UserID:          "zero-alloc-check",
		IP:              "203.0.113.88",
		FilterWorkerIdx: 0,
	}
	evt.ClickIDBuf[0] = 'c'
	evt.ClickID = unsafeString(evt.ClickIDBuf[:1])

	ctx := context.Background()
	const amount = int64(10_000)
	for range 100 {
		ledger.Credit(campID, amount, testQuotaChunkMicro)
		_ = f.Check(ctx, evt)
	}
	allocs := testing.AllocsPerRun(100, func() {
		ledger.Credit(campID, amount, testQuotaChunkMicro)
		_ = f.Check(ctx, evt)
	})
	if allocs != 0 {
		t.Fatalf("Check local-quanta full-skip allocs = %v, want 0", allocs)
	}
}
