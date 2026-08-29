package campaign

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSmartPacingExpectedRatio_daypartWeighted(t *testing.T) {
	t.Parallel()
	var weights [24]float64
	for h := 9; h <= 17; h++ {
		weights[h] = 1.0
	}
	for h := range 24 {
		if weights[h] == 0 {
			weights[h] = 0.01
		}
	}

	loc := time.UTC
	daypart := []int16{9, 10, 11, 12, 13, 14, 15, 16, 17}
	midday := time.Date(2026, 7, 7, 13, 30, 0, 0, loc)

	ratio := SmartPacingExpectedRatio(weights, daypart, midday)
	assert.InDelta(t, 0.5, ratio, 0.15)

	linear := SmartPacingExpectedRatio(UniformHourWeights(), nil, midday)
	assert.InDelta(t, 0.5625, linear, 0.02)
}
