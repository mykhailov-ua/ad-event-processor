package campaign

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCountStatusTotalsFromItems_groupsByStatus(t *testing.T) {
	t.Parallel()

	totals := CountStatusTotalsFromItems([]CampaignDTO{
		{Status: "ACTIVE"},
		{Status: "active"},
		{Status: "PAUSED"},
		{Status: "ARCHIVED"},
		{Status: "EXHAUSTED"},
	})
	assert.Equal(t, int64(2), totals.Active)
	assert.Equal(t, int64(1), totals.Paused)
	assert.Equal(t, int64(1), totals.Archived)
	assert.Equal(t, int64(5), totals.Total)
}

func TestFilterCampaignsByQuery_nameAndPacing(t *testing.T) {
	t.Parallel()

	items := []CampaignDTO{
		{ID: "1", Name: "Alpha", PacingMode: "even"},
		{ID: "2", Name: "Beta", PacingMode: "asap"},
	}
	filtered := FilterCampaignsByQuery(items, "alp", "even")
	assert.Len(t, filtered, 1)
	assert.Equal(t, "1", filtered[0].ID)
}
