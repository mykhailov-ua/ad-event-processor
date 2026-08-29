package ingest

import (
	"context"
	"os/exec"
	"strings"
	"sync/atomic"
	"testing"

	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
	redis "github.com/redis/go-redis/v9"
)

type placementHExistsMock struct {
	mockRedisClient
	hit   bool
	calls atomic.Int32
}

func (m *placementHExistsMock) HExists(ctx context.Context, key string, field string) *redis.BoolCmd {
	m.calls.Add(1)
	staticBoolCmd.SetVal(m.hit)
	return staticBoolCmd
}

func setupPlacementBlacklistBench(t testing.TB, blacklisted bool) (*PlacementBlacklistFilter, *domain.Event, context.Context) {
	t.Helper()
	redisShards := []redis.UniversalClient{&placementHExistsMock{hit: blacklisted}}
	f := NewPlacementBlacklistFilter(redisShards)
	campID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	evt := &domain.Event{
		CampaignID:  campID,
		PlacementID: "zone-42",
	}
	ctx := context.Background()
	for range 1000 {
		_ = f.Check(ctx, evt)
	}
	return f, evt, ctx
}

func BenchmarkPlacementBlacklistFilter_miss(b *testing.B) {
	f, evt, ctx := setupPlacementBlacklistBench(b, false)
	b.ReportAllocs()
	for b.Loop() {
		_ = f.Check(ctx, evt)
	}
}

func BenchmarkPlacementBlacklistFilter_hit(b *testing.B) {
	f, evt, ctx := setupPlacementBlacklistBench(b, true)
	b.ReportAllocs()
	for b.Loop() {
		_ = f.Check(ctx, evt)
	}
}

func TestPlacementBlacklistFilter_steadyStateNoHEXISTS(t *testing.T) {
	mock := &placementHExistsMock{hit: false}
	f := NewPlacementBlacklistFilter([]redis.UniversalClient{mock})
	campID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	evt := &domain.Event{
		CampaignID:  campID,
		PlacementID: "zone-42",
	}
	ctx := context.Background()

	if err := f.Check(ctx, evt); err != nil {
		t.Fatalf("first Check: %v", err)
	}
	if got := mock.calls.Load(); got != 1 {
		t.Fatalf("first Check: HEXISTS calls = %d, want 1", got)
	}

	for range 100 {
		if err := f.Check(ctx, evt); err != nil {
			t.Fatalf("steady-state Check: %v", err)
		}
	}
	if got := mock.calls.Load(); got != 1 {
		t.Fatalf("steady-state: HEXISTS calls = %d, want 1 (cache hit)", got)
	}
}

func TestPlacementBlacklistFilter_zeroAlloc(t *testing.T) {
	f, evt, ctx := setupPlacementBlacklistBench(t, false)
	avg := testing.AllocsPerRun(100, func() {
		_ = f.Check(ctx, evt)
	})
	if avg > 0 {
		t.Fatalf("PlacementBlacklistFilter.Check allocated %.1f times per run, want 0", avg)
	}
}

func TestPlacementBlacklistFilter_escapeClean(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: escape analysis bench (run make test-integration)")
	}
	root := moduleRootAds(t)
	cmd := exec.Command("go", "build", "-gcflags=-m", "-o", "/dev/null", "./internal/ingest")
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("escape analysis build failed: %v\n%s", err, out)
	}
	for _, line := range strings.Split(string(out), "\n") {
		if !strings.Contains(line, "PlacementBlacklistFilter).Check") {
			continue
		}
		if strings.Contains(line, "escapes to heap") {
			t.Fatalf("PlacementBlacklistFilter.Check escapes to heap: %s", strings.TrimSpace(line))
		}
	}
}
