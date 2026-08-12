package adminapi

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCalcCTR_invariants(t *testing.T) {
	t.Parallel()
	assert.Equal(t, float64(0), calcCTR(10, 0))
	assert.InDelta(t, 0.05, calcCTR(50, 1000), 1e-9)
	assert.True(t, calcCTR(100, 100) <= 1)
}

func TestCalcROIPct_invariants(t *testing.T) {
	t.Parallel()
	// profit = revenue - spend = 60M - 50M = 10M; ROI = 10/50*100 = 20
	assert.InDelta(t, 20, calcROIPct(10_000_000, 50_000_000), 1e-9)
	assert.Equal(t, float64(0), calcROIPct(1, 0))
}

func TestCalcIVTRate_invariants(t *testing.T) {
	t.Parallel()
	assert.Equal(t, float64(0), calcIVTRate(5, 0))
	assert.InDelta(t, 0.1, calcIVTRate(10, 100), 1e-9)
	assert.Equal(t, float64(1), calcIVTRate(200, 100)) // clamped to 1
	assert.True(t, calcIVTRate(3, 10) >= 0 && calcIVTRate(3, 10) <= 1)
}

func TestCalcCPAMicro_invariants(t *testing.T) {
	t.Parallel()
	assert.Equal(t, int64(5_000_000), calcCPAMicro(50_000_000, 10))
	assert.Equal(t, int64(0), calcCPAMicro(50, 0))
}

func TestTrueROIColumns_math(t *testing.T) {
	t.Parallel()
	adSpend := int64(100_000_000)
	revenue := int64(150_000_000)
	conversions := int64(10)
	profit := revenue - adSpend
	assert.Equal(t, int64(50_000_000), profit)
	assert.InDelta(t, 50.0, calcROIPct(profit, adSpend), 1e-9)
	assert.Equal(t, int64(10_000_000), calcCPAMicro(adSpend, conversions))
}

func TestCalcQualityFromDrift_monotone(t *testing.T) {
	t.Parallel()
	q0 := calcQualityFromDrift(0)
	q50 := calcQualityFromDrift(50)
	q120 := calcQualityFromDrift(120)
	assert.True(t, q0 >= q50 && q50 >= q120)
	assert.True(t, q0 <= 1 && q120 >= 0)
	assert.False(t, math.IsNaN(q50))
}

func TestToPlacementReportRowDTO_math(t *testing.T) {
	t.Parallel()
	row := toPlacementReportRowDTO(placementReportCHRow{
		PlacementID:  "p1",
		CampaignID:   "c1",
		Impressions:  1000,
		Clicks:       50,
		Conversions:  10,
		SpendMicro:   50_000_000,
		RevenueMicro: 60_000_000,
	}, 0)
	assert.InDelta(t, 0.05, row.CTR, 1e-9)
	assert.InDelta(t, 20, row.ROIPct, 1e-9)
	assert.Equal(t, int64(5_000_000), row.CPAMicro)
	assert.Equal(t, float64(0), row.IVTRate)
}
