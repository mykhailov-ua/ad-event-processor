package commandpalette

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMergeSearchResults_prefixBeforeSubstring(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC()
	items := mergeSearchResults(10,
		[]searchCandidate{{
			item:       ItemDTO{Label: "beta campaign"},
			prefixRank: 1,
			sortTime:   now,
		}},
		[]searchCandidate{{
			item:       ItemDTO{Label: "camp alpha"},
			prefixRank: 2,
			sortTime:   now.Add(-time.Hour),
		}},
	)
	require.Len(t, items, 2)
	assert.Equal(t, "camp alpha", items[0].Label)
}

func TestPrefixRank(t *testing.T) {
	t.Parallel()
	assert.Equal(t, 2, prefixRank("camp", "Campaign US"))
	assert.Equal(t, 1, prefixRank("camp", "US Campaign"))
	assert.Equal(t, 0, prefixRank("camp", "other"))
}

func TestParseSearchKinds_defaultsAll(t *testing.T) {
	t.Parallel()
	set := parseSearchKinds(nil)
	assert.True(t, set.searchCampaigns)
	assert.True(t, set.searchFlows)
	assert.True(t, set.searchLanders)
	assert.True(t, set.searchOffers)
}

func TestCommandPalette_emptyQuery_holdout(t *testing.T) {
	t.Parallel()
	rec := &recordingSearcher{}
	svc := &Service{Store: rec}
	resp := svc.Search(t.Context(), uuid.MustParse("00000000-0000-4000-8000-000000000001"), "", 25, nil, nil)
	assert.Empty(t, resp.Items)
	assert.False(t, resp.Degraded)
	assert.False(t, rec.called, "empty q must not trigger entity search")
}
