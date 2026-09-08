package domain

import "testing"

func TestParseCrossLayerDesyncAction(t *testing.T) {
	tests := []struct {
		in   string
		want CrossLayerDesyncAction
	}{
		{"off", CrossLayerDesyncOff},
		{"boost", CrossLayerDesyncBoost},
		{"safe_page", CrossLayerDesyncSafePage},
		{"block", CrossLayerDesyncBlock},
		{"", CrossLayerDesyncBoost},
	}
	for _, tc := range tests {
		got := ParseCrossLayerDesyncAction(tc.in)
		if got != tc.want {
			t.Fatalf("ParseCrossLayerDesyncAction(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestNormalizeCrossLayerDesyncThreshold(t *testing.T) {
	assert := func(got, want uint8) {
		if got != want {
			t.Fatalf("got %d want %d", got, want)
		}
	}
	assert(NormalizeCrossLayerDesyncThreshold(0), CrossLayerDesyncThresholdDefault)
	assert(NormalizeCrossLayerDesyncThreshold(3), 3)
	assert(NormalizeCrossLayerDesyncThreshold(9), CrossLayerDesyncThresholdMax)
}
