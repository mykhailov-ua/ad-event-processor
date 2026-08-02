package controlplane

import (
	"testing"

	"espx/internal/controlplane/adminapi"

	"github.com/stretchr/testify/assert"
)

func TestWorstSourcesFromCampaigns_qualityMonotone(t *testing.T) {
	t.Parallel()
	campaigns := []adminapi.BuyerCampaignRowDTO{
		{ID: "a", Name: "Low drift", PacingDriftPct: 10, SpendMicro: 100},
		{ID: "b", Name: "High drift", PacingDriftPct: 80, OverspendRisk: true, SpendMicro: 200},
	}
	out := worstSourcesFromCampaigns(campaigns)
	if assert.Len(t, out, 2) {
		assert.Equal(t, "b", out[0].CampaignID)
		assert.InDelta(t, adminapi.CalcQualityFromDrift(80), out[0].QualityScore, 1e-9)
		assert.Equal(t, float64(0), out[0].IVTRate)
	}
}
