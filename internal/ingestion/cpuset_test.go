package ingestion

import "testing"

func TestCount(t *testing.T) {
	tests := []struct {
		in  string
		out int
	}{
		{"", 0},
		{"0", 1},
		{"0-3", 4},
		{"0,2,4", 3},
		{"0-1,3", 3},
		{" 0 - 1 , 3 ", 3},
	}
	for _, tc := range tests {
		if got := Count(tc.in); got != tc.out {
			t.Fatalf("Count(%q)=%d want %d", tc.in, got, tc.out)
		}
	}
}
