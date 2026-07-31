package ingestion

import (
	"context"
	"os"
	"runtime"
	"testing"
	"time"

	"espx/internal/domain"
	"github.com/google/uuid"
)

func benchWritePathEvent() *domain.Event {
	return &domain.Event{
		ClickID:    "bench-click",
		CampaignID: uuid.New(),
		Type:       "click",
		IP:         "203.0.113.1",
		UA:         "bench-agent",
		Payload:    []byte(`{"bench":true}`),
		CreatedAt:  time.Unix(1_700_000_000, 0).UTC(),
	}
}

func BenchmarkCHSpoolAppendDurably(b *testing.B) {
	dir := b.TempDir()
	spool, err := OpenCHSpool(dir)
	if err != nil {
		b.Fatal(err)
	}
	defer func() { _ = spool.Close() }()

	evt := benchWritePathEvent()
	events := []*domain.Event{evt}
	token := "bench-dedup-token"

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if err := spool.AppendDurably(token, events); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCHSpoolMarshalPayload(b *testing.B) {
	evt := benchWritePathEvent()
	events := []*domain.Event{evt}
	token := "bench-dedup-token"

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := marshalCHSpoolPayload(token, events); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPostgresStoreBatch_Mock(b *testing.B) {
	store := &MockEventStore{}
	evt := benchWritePathEvent()
	events := []*domain.Event{evt}
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if err := store.StoreBatch(ctx, events); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkClickHouseStoreBatch_Spooled(b *testing.B) {
	dir := b.TempDir()
	spool, err := OpenCHSpool(dir)
	if err != nil {
		b.Fatal(err)
	}
	defer func() { _ = spool.Close() }()

	conn := newFailingCHConn(true)
	store := NewClickHouseStore(conn, time.Second, "", DefaultCHSpoolConfig(), nil)
	store.SetSpool(spool)

	evt := benchWritePathEvent()
	events := []*domain.Event{evt}
	ctx := context.WithValue(context.Background(), domain.DeduplicationTokenKey, "bench-ch-spool")

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if err := store.StoreBatch(ctx, events); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCHSpoolOpenFdDelta(b *testing.B) {
	if runtime.GOOS != "linux" {
		b.Skip("requires /proc/self/fd")
	}
	before, err := os.ReadDir("/proc/self/fd")
	if err != nil {
		b.Fatal(err)
	}
	b.Logf("fd_before=%d", len(before))

	dir := b.TempDir()
	spool, err := OpenCHSpool(dir)
	if err != nil {
		b.Fatal(err)
	}
	afterOpen, err := os.ReadDir("/proc/self/fd")
	if err != nil {
		b.Fatal(err)
	}
	b.Logf("fd_after_open=%d delta=%d", len(afterOpen), len(afterOpen)-len(before))

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = spool.WritePos()
	}
	_ = spool.Close()
}
