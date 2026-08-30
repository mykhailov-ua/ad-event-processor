package commandpalette

import (
	"context"
	"strings"
	"time"
)

type catalogKindSet struct {
	searchReports bool
	searchRoutes  bool
	searchActions bool
}

func (s catalogKindSet) any() bool {
	return s.searchReports || s.searchRoutes || s.searchActions
}

func parseCatalogKinds(kinds []string) catalogKindSet {
	if len(kinds) == 0 {
		return catalogKindSet{
			searchReports: true,
			searchRoutes:  true,
			searchActions: true,
		}
	}
	var set catalogKindSet
	for _, kind := range kinds {
		switch strings.ToLower(strings.TrimSpace(kind)) {
		case "report":
			set.searchReports = true
		case "route":
			set.searchRoutes = true
		case "action":
			set.searchActions = true
		}
	}
	return set
}

func searchCatalog(ctx context.Context, query string, limit int, kinds catalogKindSet, licenseAllowed func(string) bool) []searchCandidate {
	if !kinds.any() || limit <= 0 {
		return nil
	}
	query = truncateSearchQuery(query)
	if query == "" {
		return nil
	}
	items := FilterNavCatalog(ctx, NavCatalogEntries(), licenseAllowed)
	if len(items) == 0 {
		return nil
	}
	now := time.Now().UTC()
	out := make([]searchCandidate, 0, len(items))
	for _, item := range items {
		if !catalogKindAllowed(kinds, item.Kind) {
			continue
		}
		if !catalogItemMatchesQuery(query, item) {
			continue
		}
		out = append(out, searchCandidate{
			item:       item,
			prefixRank: catalogPrefixRank(query, item),
			sortTime:   now,
		})
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func catalogKindAllowed(kinds catalogKindSet, kind string) bool {
	switch kind {
	case "report":
		return kinds.searchReports
	case "route":
		return kinds.searchRoutes
	case "action":
		return kinds.searchActions
	default:
		return false
	}
}

func catalogItemMatchesQuery(query string, item ItemDTO) bool {
	lowerQ := strings.ToLower(strings.TrimSpace(query))
	if lowerQ == "" {
		return false
	}
	for _, field := range []string{item.Label, item.Meta, item.Href, strings.TrimPrefix(item.Href, "/")} {
		if field != "" && strings.Contains(strings.ToLower(field), lowerQ) {
			return true
		}
	}
	if strings.HasPrefix(item.ID, "report:") {
		key := strings.TrimPrefix(item.ID, "report:")
		if strings.Contains(strings.ToLower(key), lowerQ) {
			return true
		}
	}
	return false
}

func catalogPrefixRank(query string, item ItemDTO) int {
	best := prefixRank(query, item.Label)
	if metaRank := prefixRank(query, item.Meta); metaRank > best {
		best = metaRank
	}
	if key := strings.TrimPrefix(item.ID, "report:"); key != item.ID {
		if keyRank := prefixRank(query, key); keyRank > best {
			best = keyRank
		}
	}
	return best
}

func dtoItemsToCandidates(query string, items []ItemDTO) []searchCandidate {
	out := make([]searchCandidate, 0, len(items))
	for _, item := range items {
		out = append(out, searchCandidate{
			item:       item,
			prefixRank: prefixRank(query, item.Label),
		})
	}
	return out
}
