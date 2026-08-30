package campaign

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFilterAndSortCampaigns_queryAndSort(t *testing.T) {
	t.Parallel()
	items := []CampaignDTO{
		{ID: "b", Name: "Beta", UpdatedAt: "2026-01-02T00:00:00Z"},
		{ID: "a", Name: "Alpha", UpdatedAt: "2026-01-01T00:00:00Z"},
	}
	got := filterAndSortCampaigns(items, "alp", "name", "asc", "")
	require.Len(t, got, 1)
	assert.Equal(t, "Alpha", got[0].Name)
}
