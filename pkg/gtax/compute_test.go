package gtax

import "testing"

func TestComputeMicro(t *testing.T) {
	t.Parallel()
	tests := []struct {
		spend int64
		bps   int32
		want  int64
	}{
		{0, 500, 0},
		{1_000_000, 0, 0},
		{1_000_000, 500, 50_000},
		{10_000_000, 725, 725_000},
	}
	for _, tc := range tests {
		got := ComputeMicro(tc.spend, tc.bps)
		if got != tc.want {
			t.Fatalf("ComputeMicro(%d,%d)=%d want %d", tc.spend, tc.bps, got, tc.want)
		}
	}
}
