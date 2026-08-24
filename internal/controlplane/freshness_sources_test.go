package controlplane

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPortfolioFreshness_withoutCH(t *testing.T) {
	t.Parallel()
	dto := portfolioFreshness(time.Unix(0, 0).UTC(), false, 0)
	assert.Equal(t, "strong", dto.Consistency)
	assert.True(t, dto.Stale)
	require.Len(t, dto.Sources, 1)
	assert.Equal(t, "counts", dto.Sources[0].Name)
	assert.Equal(t, "strong", dto.Sources[0].Consistency)
}

func TestPortfolioFreshness_withCHLag(t *testing.T) {
	t.Parallel()
	dto := portfolioFreshness(time.Unix(0, 0).UTC(), true, 10*time.Minute)
	assert.Equal(t, "mixed", dto.Consistency)
	assert.True(t, dto.Stale)
	require.Len(t, dto.Sources, 2)
	assert.Equal(t, "money", dto.Sources[1].Name)
	assert.Equal(t, "eventual", dto.Sources[1].Consistency)
}

func TestCampaignDashboardFreshness_CHMoney(t *testing.T) {
	t.Parallel()
	now := time.Unix(0, 0).UTC()
	dto := campaignDashboardFreshness(now, true, 2*time.Minute, true)
	assert.Equal(t, "mixed", dto.Consistency)
	require.Len(t, dto.Sources, 2)
	assert.Equal(t, "counts", dto.Sources[0].Name)
	assert.Equal(t, "money", dto.Sources[1].Name)
	assert.Equal(t, "eventual", dto.Sources[1].Consistency)
}

func TestCampaignDashboardFreshness_PGOnly(t *testing.T) {
	t.Parallel()
	now := time.Unix(0, 0).UTC()
	dto := campaignDashboardFreshness(now, false, 0, false)
	assert.Equal(t, "strong", dto.Consistency)
	require.Len(t, dto.Sources, 2)
	assert.Equal(t, "strong", dto.Sources[1].Consistency)
}
