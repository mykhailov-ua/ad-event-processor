package main

import "testing"

func TestChooseTrackerBinary_prefersProcExe(t *testing.T) {
	t.Parallel()
	got := chooseTrackerBinary("/proc/1234/exe", "/host/bin/tracker-bpf-trace")
	want := "/proc/1234/exe"
	if got != want {
		t.Fatalf("chooseTrackerBinary() = %q, want %q", got, want)
	}
}

func TestChooseTrackerBinary_fallsBackToFlag(t *testing.T) {
	t.Parallel()
	got := chooseTrackerBinary("", "/host/bin/tracker-bpf-trace")
	if got != "/host/bin/tracker-bpf-trace" {
		t.Fatalf("chooseTrackerBinary() = %q, want host flag fallback", got)
	}
}
