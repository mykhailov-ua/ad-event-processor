package ingestion

import (
	"context"
	"os"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/bidshard/ad-event-processor/internal/database"
	"github.com/bidshard/ad-event-processor/internal/domain"
	db "github.com/bidshard/ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
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

// BenchmarkPostgresStoreBatch_Mock measures StoreBatch against MockEventStore only.
// Harness: mock_event_store — mock store only; not Postgres.
// For Postgres write latency use BenchmarkPostgresStoreBatch_integration or processor
// integration tests (e.g. TestFault_ProcessorPgGate_Overflow, TestFault_AdsProcessorPGNetworkPartition).
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

// BenchmarkClickHouseStoreBatch_Spooled measures CH StoreBatch on the local spool path (CH insert fails).
// Harness: ch_spool_local. Use for cold-path CH outage write perf; not Postgres.
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

func setupPostgresStoreBench(b *testing.B) (*PostgresStore, func()) {
	ctx := context.Background()
	pgContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("write_path_bench_db"),
		postgres.WithUsername("user"),
		postgres.WithPassword("pass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(20*time.Second)),
	)
	if err != nil {
		b.Fatalf("failed to start postgres container: %s", err)
	}

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		b.Fatalf("postgres connection string: %s", err)
	}
	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		b.Fatalf("postgres pool: %s", err)
	}
	applyAdsMigrations(b, pool)

	pm := database.NewPartitionManager(pool, 7, 1)
	if err := pm.Run(ctx); err != nil {
		b.Fatalf("partition manager: %s", err)
	}

	store := NewPostgresStore(db.New(pool), 2*time.Second)
	cleanup := func() {
		pool.Close()
		_ = pgContainer.Terminate(ctx)
	}
	return store, cleanup
}

// BenchmarkPostgresStoreBatch_integration runs StoreBatch against testcontainers Postgres.
// Harness: postgres_testcontainers. Skipped with -short. Not in make test-alloc-gate;
// use scripts/test/run_bench.sh (see docs/DEVELOPMENT.md perf section).
func BenchmarkPostgresStoreBatch_integration(b *testing.B) {
	if testing.Short() {
		b.Skip()
	}
	store, cleanup := setupPostgresStoreBench(b)
	defer cleanup()

	evt := benchWritePathEvent()
	events := []*domain.Event{evt}
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		evt.ClickID = "bench-pg-" + strconv.Itoa(i)
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
