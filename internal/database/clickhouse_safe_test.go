package database

import "testing"

func TestValidClickHouseIdentifier(t *testing.T) {
	tests := []struct {
		name string
		in   string
		ok   bool
	}{
		{name: "impressions", in: "impressions", ok: true},
		{name: "partition", in: "202401", ok: true},
		{name: "empty", in: "", ok: false},
		{name: "quote", in: "foo'; DROP", ok: false},
		{name: "space", in: "foo bar", ok: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := ValidClickHouseIdentifier(tc.in); got != tc.ok {
				t.Fatalf("ValidClickHouseIdentifier(%q)=%v want %v", tc.in, got, tc.ok)
			}
		})
	}
}

func TestClampCHLookbackHours(t *testing.T) {
	if got := ClampCHLookbackHours(0); got != 1 {
		t.Fatalf("zero=%d", got)
	}
	if got := ClampCHLookbackHours(24 * 100); got != 24*90 {
		t.Fatalf("max=%d", got)
	}
}

func TestClampCHWindowSeconds(t *testing.T) {
	if got := ClampCHWindowSeconds(0); got != 3600 {
		t.Fatalf("zero=%d", got)
	}
	if got := ClampCHWindowSeconds(48 * 3600); got != 24*3600 {
		t.Fatalf("max=%d", got)
	}
}

func TestValidGAQLDate(t *testing.T) {
	if !ValidGAQLDate("2026-01-15") {
		t.Fatal("expected valid date")
	}
	if ValidGAQLDate("2026-13-01") {
		t.Fatal("expected invalid month")
	}
}
