package reports

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildCustomerFraudOverview_staleFreshness(t *testing.T) {
	t.Parallel()
	out := BuildCustomerFraudOverview(0, 0, 0, DataFreshnessDTO{Stale: true, CHLagSeconds: 400})
	assert.True(t, out.Freshness.Stale)
	assert.NotEmpty(t, out.Disclaimer)
}

func TestBuildCustomerFraudOverview_ratesAndDisplay(t *testing.T) {
	t.Parallel()
	out := BuildCustomerFraudOverview(100, 25, 10, DataFreshnessDTO{Stale: true})
	assert.Equal(t, int64(100), out.TotalEvents)
	assert.InDelta(t, 0.25, out.BlockRate, 0.0001)
	assert.Equal(t, "25.0%", out.BlockRateDisplay)
	assert.Equal(t, "10.0%", out.SilentRejectRateDisplay)
	assert.NotEmpty(t, out.Disclaimer)
}
