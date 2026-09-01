package coldpath

import "testing"

func TestRFC3339Display_parsesNanoAndUTC(t *testing.T) {
	got := RFC3339Display("2026-02-01T10:30:45.123456789Z")
	if got != "2026-02-01 10:30 UTC" {
		t.Fatalf("RFC3339Display() = %q, want 2026-02-01 10:30 UTC", got)
	}
}

func TestRFC3339Display_unparseablePassthrough(t *testing.T) {
	raw := "not-a-timestamp"
	if got := RFC3339Display(raw); got != raw {
		t.Fatalf("RFC3339Display() = %q, want passthrough %q", got, raw)
	}
}
