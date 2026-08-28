package ingestion

import (
	"os"
	"runtime"
	"testing"
)

func TestDefaultMaxWorkersFromCPU(t *testing.T) {
	_ = os.Unsetenv("MAX_WORKERS")
	want := runtime.NumCPU()
	if want < 1 {
		want = 1
	}
	if got := DefaultMaxWorkers(); got != want {
		t.Fatalf("DefaultMaxWorkers=%d want %d", got, want)
	}
}

func TestDefaultMaxWorkersRespectsExplicitEnv(t *testing.T) {
	t.Setenv("MAX_WORKERS", "4")
	if got := DefaultMaxWorkers(); got != 0 {
		t.Fatalf("DefaultMaxWorkers=%d want 0 when MAX_WORKERS set", got)
	}
}

func TestApplyRuntimeAutotuneGOMAXPROCSFromTrackerCpuset(t *testing.T) {
	_ = os.Unsetenv("GOMAXPROCS")
	t.Setenv("TRACKER_CPUSET", "0-3")
	before := runtime.GOMAXPROCS(0)
	ApplyRuntimeAutotune()
	if got := runtime.GOMAXPROCS(0); got != 4 {
		t.Fatalf("GOMAXPROCS=%d want 4 from TRACKER_CPUSET", got)
	}
	runtime.GOMAXPROCS(before)
}
