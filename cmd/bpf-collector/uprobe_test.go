package main

import "testing"

func TestChooseTrackerBinary_prefersRunningExe(t *testing.T) {
	t.Parallel()
	got := chooseTrackerBinary("/var/lib/docker/overlay2/merged/tracker", "/host/bin/tracker-bpf-trace")
	want := "/var/lib/docker/overlay2/merged/tracker"
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
