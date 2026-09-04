package commandpalette

import (
	"fmt"
	"testing"
	"time"
)

func syntheticSearchCandidates(n int) []searchCandidate {
	out := make([]searchCandidate, n)
	for i := range out {
		out[i] = searchCandidate{
			item: ItemDTO{
				Kind:  "route",
				Label: fmt.Sprintf("Report %d campaign performance", i),
				Href:  fmt.Sprintf("/reports/r-%d", i),
			},
			prefixRank: 1,
			sortTime:   time.Now().Add(-time.Duration(i) * time.Minute),
		}
	}
	return out
}

func BenchmarkPrefixRank(b *testing.B) {
	query := "camp"
	name := "Campaign performance overview"
	b.ResetTimer()
	for b.Loop() {
		prefixRank(query, name)
	}
}

func BenchmarkMergeSearchResults_1k(b *testing.B) {
	batchA := syntheticSearchCandidates(500)
	batchB := syntheticSearchCandidates(500)
	b.ResetTimer()
	for b.Loop() {
		mergeSearchResults(DefaultSearchLimit, batchA, batchB)
	}
}

func BenchmarkCatalogItemMatchesQuery(b *testing.B) {
	query := "revenue"
	item := ItemDTO{
		Kind:  "report",
		Label: "Revenue by campaign",
		Meta:  "campaigns",
		ID:    "report:revenue-by-campaign",
	}
	b.ResetTimer()
	for b.Loop() {
		catalogItemMatchesQuery(query, item)
	}
}

func TestMergeSearchResults_respectsLimit_holdout(t *testing.T) {
	batches := []searchCandidate{
		{item: ItemDTO{Label: "a"}, prefixRank: 2},
		{item: ItemDTO{Label: "b"}, prefixRank: 2},
		{item: ItemDTO{Label: "c"}, prefixRank: 1},
	}
	got := mergeSearchResults(2, batches)
	if len(got) != 2 {
		t.Fatalf("expected limit 2, got %d", len(got))
	}
}
