package campaign

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestDeriveCampaignPacingHealth_exhaustedOnBudget(t *testing.T) {
	meta := campaignOperationalMeta{
		status:       "ACTIVE",
		budgetLimit:  1_000_000,
		currentSpend: 1_000_000,
		pacingMode:   "EVEN",
	}
	require.Equal(t, "exhausted", deriveCampaignPacingHealth(meta))
}

func TestDeriveCampaignPacingHealth_driftOnEvenMode(t *testing.T) {
	operationalNow = func() time.Time {
		return time.Date(2026, 3, 11, 6, 0, 0, 0, time.UTC)
	}
	t.Cleanup(func() { operationalNow = time.Now })

	meta := campaignOperationalMeta{
		status:          "ACTIVE",
		budgetLimit:     10_000_000,
		currentSpend:    1_000_000,
		pacingMode:      "EVEN",
		dailyBudget:     1_000_000,
		timezone:        "UTC",
		todaySpendMicro: 500_000,
	}
	require.Equal(t, "drift", deriveCampaignPacingHealth(meta))
}

func TestDeriveCampaignPacingHealth_okOnAsap(t *testing.T) {
	meta := campaignOperationalMeta{
		status:          "ACTIVE",
		pacingMode:      "ASAP",
		dailyBudget:     1_000_000,
		todaySpendMicro: 900_000,
	}
	require.Equal(t, "ok", deriveCampaignPacingHealth(meta))
}

func TestCampaignBudgetBurnPct_holdoutZeroBudget(t *testing.T) {
	require.Equal(t, float64(0), campaignBudgetBurnPct(500, 0))
}

func TestOperationalPacingExpectedRatio_midday(t *testing.T) {
	now := time.Date(2026, 3, 11, 12, 0, 0, 0, time.UTC)
	ratio := operationalPacingExpectedRatio(now)
	require.InDelta(t, 0.5, ratio, 0.02)
}
