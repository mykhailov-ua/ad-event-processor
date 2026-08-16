package runtimeautotune_test

import (
	"os"
	"runtime"
	"testing"

	"github.com/bidshard/ad-event-processor/internal/config"
	"github.com/bidshard/ad-event-processor/pkg/runtimeautotune"
)

func TestApplyMaxWorkersFromCPU(t *testing.T) {
	_ = os.Unsetenv("MAX_WORKERS")
	cfg := &config.Config{MaxWorkers: 16}
	runtimeautotune.Apply(cfg)
	want := runtime.NumCPU()
	if want < 1 {
		want = 1
	}
	if cfg.MaxWorkers != want {
		t.Fatalf("MaxWorkers=%d want %d", cfg.MaxWorkers, want)
	}
}

func TestApplyRespectsExplicitMaxWorkers(t *testing.T) {
	t.Setenv("MAX_WORKERS", "4")
	cfg := &config.Config{MaxWorkers: 16}
	runtimeautotune.Apply(cfg)
	if cfg.MaxWorkers != 16 {
		t.Fatalf("MaxWorkers=%d want unchanged 16", cfg.MaxWorkers)
	}
}

func TestApplyGOMAXPROCSFromTrackerCpuset(t *testing.T) {
	_ = os.Unsetenv("GOMAXPROCS")
	t.Setenv("TRACKER_CPUSET", "0-3")
	before := runtime.GOMAXPROCS(0)
	runtimeautotune.Apply(&config.Config{})
	if got := runtime.GOMAXPROCS(0); got != 4 {
		t.Fatalf("GOMAXPROCS=%d want 4 from TRACKER_CPUSET", got)
	}
	runtime.GOMAXPROCS(before)
}
