package fraud

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFeatureRowToVectorDerived(t *testing.T) {
	row := FeatureRow{
		Events:           200,
		Clicks:           180,
		SpendMicro:       50_000_000,
		BudgetLimitMicro: 60_000_000,
		UniqueUsers:      2,
		UniqueUAs:        1,
	}
	got := row.ToVector()
	assert.Len(t, got, Dims())
	assert.InDelta(t, 200.0, got[0], 1e-9)
	assert.InDelta(t, 0.9, got[2], 1e-9)
	assert.InDelta(t, 200.0, got[7], 1e-9)
	assert.InDelta(t, 180.0, got[8], 1e-9)
	assert.InDelta(t, 2.0, got[9], 1e-9)
	assert.InDelta(t, 90.0, got[10], 1e-9)
	assert.InDelta(t, 50.0/180.0, got[11], 1e-9)
	assert.InDelta(t, 1.0/200.0, got[12], 1e-9)
	assert.InDelta(t, 100.0, got[13], 1e-9)
	assert.InDelta(t, 200.0/181.0, got[14], 1e-9)
	assert.InDelta(t, 2.0/181.0, got[15], 1e-9)
}

func TestFeatureRowToVectorZeroSafe(t *testing.T) {
	got := (&FeatureRow{}).ToVector()
	assert.Len(t, got, Dims())
	for i := range got {
		assert.InDelta(t, 0.0, got[i], 1e-9, "index %d", i)
	}
}
