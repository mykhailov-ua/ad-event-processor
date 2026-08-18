package ingestion

import (
	"context"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/bidshard/ad-event-processor/internal/config"
	"github.com/bidshard/ad-event-processor/internal/domain"

	"github.com/google/uuid"
)

func setupFilterFraudBoostBench(t testing.TB) (*FilterEngine, *domain.Event, context.Context) {
	t.Helper()
	cfg := &config.Config{}
	sw := NewSettingsWatcher(nil, cfg)
	campID := uuid.New()
	sw.fraudScoreBoosts.Store(&FraudScoreBoostSnapshot{
		Boosts: map[uuid.UUID]uint8{campID: 15},
	})

	engine := NewFilterEngine(0, &fraudSignalsFilter{first: FraudReasonMissingImpTS})
	engine.SetRegistry(&mockRegistry{})
	engine.SetSettingsWatcher(sw)

	cachedMockCamp.Store(&domain.Campaign{ID: campID})
	t.Cleanup(func() { cachedMockCamp.Store(nil) })

	evt := &domain.Event{
		CampaignID:   campID,
		StringBuffer: make([]byte, 0, 64),
	}
	ctx := context.Background()

	for range 1000 {
		resetFraudBenchEvent(evt)
		_ = engine.Check(ctx, evt)
	}
	return engine, evt, ctx
}

func BenchmarkFilterFraudBoost(b *testing.B) {
	engine, evt, ctx := setupFilterFraudBoostBench(b)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resetFraudBenchEvent(evt)
		_ = engine.Check(ctx, evt)
	}
}

func TestFilterFraudBoost_zeroAlloc(t *testing.T) {
	engine, evt, ctx := setupFilterFraudBoostBench(t)
	for range 100 {
		resetFraudBenchEvent(evt)
		_ = engine.Check(ctx, evt)
	}
	avg := testing.AllocsPerRun(100, func() {
		resetFraudBenchEvent(evt)
		_ = engine.Check(ctx, evt)
	})
	if avg > 0 {
		t.Fatalf("FilterEngine.Check with ML boost allocated %.1f times per run, want 0", avg)
	}
}

func moduleRootAds(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func TestFilterFraudBoost_escapeClean(t *testing.T) {
	if testing.Short() {
		t.Skip("escape analysis build")
	}
	root := moduleRootAds(t)
	cmd := exec.Command("go", "build", "-gcflags=-m", "-o", osDevNull(), "./internal/ingestion")
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("escape analysis build failed: %v\n%s", err, out)
	}
	text := string(out)
	for _, line := range strings.Split(text, "\n") {
		if !strings.Contains(line, "applyFraudLayerDecision") {
			continue
		}
		if strings.Contains(line, "escapes to heap") {
			t.Fatalf("applyFraudLayerDecision escapes to heap: %s", strings.TrimSpace(line))
		}
	}
}

func osDevNull() string {
	return "/dev/null"
}
