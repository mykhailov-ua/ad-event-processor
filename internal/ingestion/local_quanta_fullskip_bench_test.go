//go:build !race

package ingestion

import (
	"context"
	"strconv"
	"testing"
	"time"

	"espx/internal/domain"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	redis "github.com/redis/go-redis/v9"
)

func BenchmarkLocalQuanta_FullSkip(b *testing.B) {
	mr, err := miniredis.Run()
	if err != nil {
		b.Fatal(err)
	}
	defer mr.Close()
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})

	reg := &mockRegistry{}
	f := NewUnifiedFilter(
		[]redis.UniversalClient{rdb},
		NewJumpHashSharder(1),
		reg,
		nil,
		0,
		time.Minute,
		time.Hour,
		time.Hour,
		100,
		10,
		"events",
		1000,
	)
	f.SetQuotaConfig("live", testQuotaChunkMicro, testQuotaRefillThreshold)
	f.SetLuaFastPathEnabled(true)
	f.SetTTCMin(0)
	ledger := NewLocalQuantaLedger()
	idem := NewLocalClickIdemCache(time.Hour)
	stream := NewLocalQuantaStreamPublisher(LocalQuantaStreamPublisherConfig{
		Rdbs:           []redis.UniversalClient{rdb},
		StreamName:     "events",
		MaxLen:         1000,
		IdempotencyTTL: time.Hour,
		IdemCache:      idem,
	})
	defer stream.Close()
	f.SetLocalQuantaDeps(LocalQuantaDeps{Ledger: ledger, Stream: stream})
	f.SetLocalQuantaMode("live")

	campID := uuid.New()
	ledger.Credit(campID, int64(b.N)*f.clickAmountMicro+testQuotaChunkMicro, testQuotaChunkMicro)

	ctx := attachFilterDeadline(context.Background(), time.Minute)
	evt := &domain.Event{
		Type:       "click",
		IP:         "203.0.113.70",
		UserID:     "bench-full-skip",
		CampaignID: campID,
	}
	clickScratch := evt.ClickIDBuf[:0]
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf := strconv.AppendInt(clickScratch[:0], int64(i), 10)
		if len(buf) > len(evt.ClickIDBuf) {
			b.Fatal("click id overflow")
		}
		copy(evt.ClickIDBuf[:], buf)
		evt.ClickID = unsafeString(evt.ClickIDBuf[:len(buf)])
		_ = f.Check(ctx, evt)
	}
}
