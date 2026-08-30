package commandpalette

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	DefaultSearchLimit = 25
	MaxSearchLimit     = 25
	MaxSearchQueryLen  = 128
	MinSearchQueryLen  = 2
	searchTimeout      = 500 * time.Millisecond
)

type EntitySearcher interface {
	SearchEntities(ctx context.Context, customerID uuid.UUID, query string, limit int, kinds []string) ([]ItemDTO, error)
}

type Service struct {
	Store   EntitySearcher
	Recents *RecentsStore
}

func NewService(pool *pgxpool.Pool) *Service {
	if pool == nil {
		return &Service{}
	}
	return &Service{
		Store:   NewStore(pool),
		Recents: NewRecentsStore(pool),
	}
}

func (s *Service) Search(ctx context.Context, customerID uuid.UUID, query string, limit int, kinds []string, licenseAllowed func(string) bool) SearchResponse {
	if limit <= 0 {
		limit = DefaultSearchLimit
	}
	if limit > MaxSearchLimit {
		limit = MaxSearchLimit
	}
	query = normalizeSearchQuery(query)
	if query == "" {
		return SearchResponse{
			Items:          []ItemDTO{},
			Total:          0,
			Limit:          limit,
			Degraded:       false,
			FreshnessLabel: "Now",
		}
	}
	catalogKinds := parseCatalogKinds(kinds)
	entityKinds := parseSearchKinds(kinds)
	var batches [][]searchCandidate
	if catalogKinds.any() {
		if batch := searchCatalog(ctx, query, limit, catalogKinds, licenseAllowed); len(batch) > 0 {
			batches = append(batches, batch)
		}
	}
	degraded := false
	if entityKinds.any() {
		if s == nil || s.Store == nil {
			degraded = true
		} else {
			searchCtx, cancel := context.WithTimeout(ctx, searchTimeout)
			entityItems, err := s.Store.SearchEntities(searchCtx, customerID, query, limit, kinds)
			cancel()
			if err != nil {
				degraded = true
			} else if len(entityItems) > 0 {
				batches = append(batches, dtoItemsToCandidates(query, entityItems))
			}
		}
	}
	if len(batches) == 0 {
		if degraded {
			return degradedSearchResponse(limit)
		}
		return SearchResponse{
			Items:          []ItemDTO{},
			Total:          0,
			Limit:          limit,
			Degraded:       false,
			FreshnessLabel: "Now",
		}
	}
	items := mergeSearchResults(limit, batches...)
	return SearchResponse{
		Items:          items,
		Total:          int64(len(items)),
		Limit:          limit,
		Degraded:       degraded,
		FreshnessLabel: "Now",
	}
}

func degradedSearchResponse(limit int) SearchResponse {
	return SearchResponse{
		Items:          []ItemDTO{},
		Total:          0,
		Limit:          limit,
		Degraded:       true,
		FreshnessLabel: "Now",
	}
}

func ParseSearchKinds(raw []string) []string {
	if len(raw) == 0 {
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, kind := range raw {
		kind = strings.ToLower(strings.TrimSpace(kind))
		if kind == "" {
			continue
		}
		out = append(out, kind)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
