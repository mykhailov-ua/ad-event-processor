package commandpalette

import (
	"sort"
	"strings"
	"time"
)

type searchCandidate struct {
	item       ItemDTO
	prefixRank int
	sortTime   time.Time
}

func mergeSearchResults(limit int, batches ...[]searchCandidate) []ItemDTO {
	if limit <= 0 {
		limit = DefaultSearchLimit
	}
	merged := make([]searchCandidate, 0, limit*len(batches))
	for _, batch := range batches {
		merged = append(merged, batch...)
	}
	sort.SliceStable(merged, func(i, j int) bool {
		left := merged[i]
		right := merged[j]
		if left.prefixRank != right.prefixRank {
			return left.prefixRank > right.prefixRank
		}
		if !left.sortTime.Equal(right.sortTime) {
			return left.sortTime.After(right.sortTime)
		}
		return strings.ToLower(left.item.Label) < strings.ToLower(right.item.Label)
	})
	if len(merged) > limit {
		merged = merged[:limit]
	}
	out := make([]ItemDTO, len(merged))
	for i := range merged {
		out[i] = merged[i].item
	}
	return out
}

func prefixRank(query, name string) int {
	lowerQ := strings.ToLower(strings.TrimSpace(query))
	lowerN := strings.ToLower(strings.TrimSpace(name))
	if lowerQ == "" || lowerN == "" {
		return 0
	}
	if strings.HasPrefix(lowerN, lowerQ) {
		return 2
	}
	if strings.Contains(lowerN, lowerQ) {
		return 1
	}
	return 0
}

func normalizeSearchQuery(query string) string {
	return strings.TrimSpace(query)
}

func truncateSearchQuery(query string) string {
	query = normalizeSearchQuery(query)
	if len(query) > MaxSearchQueryLen {
		return query[:MaxSearchQueryLen]
	}
	return query
}
