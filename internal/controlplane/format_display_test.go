package controlplane

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatMicrosDisplay(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "USD 1.23", formatMicrosDisplay(1_230_000, "USD"))
}

func TestFormatMicrosDisplay_zero(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "USD 0.00", formatMicrosDisplay(0, "USD"))
}

func TestFormatDeltaLabel_negative(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "-5.0% vs prior period", formatDeltaLabel(-5))
}

func TestFormatRateDisplay(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "12.3%", formatRateDisplay(0.123))
}

func TestFormatBasisPointsDisplay_holdout(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "5.0%", formatBasisPointsDisplay(500))
	assert.Equal(t, "2.5%", formatBasisPointsDisplay(250))
}

func TestDeltaTone_holdout(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "positive", deltaTone(0.5))
	assert.Equal(t, "negative", deltaTone(-0.1))
	assert.Equal(t, "neutral", deltaTone(0))
}
