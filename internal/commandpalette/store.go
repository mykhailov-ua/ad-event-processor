package commandpalette

import (
	"context"
	"fmt"
	"strings"
	"time"

	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

const perKindSearchBudget = 10

type paletteQuerier interface {
	SearchCommandPaletteCampaigns(ctx context.Context, arg db.SearchCommandPaletteCampaignsParams) ([]db.SearchCommandPaletteCampaignsRow, error)
	SearchCommandPaletteFlows(ctx context.Context, arg db.SearchCommandPaletteFlowsParams) ([]db.SearchCommandPaletteFlowsRow, error)
	SearchCommandPaletteLanders(ctx context.Context, arg db.SearchCommandPaletteLandersParams) ([]db.SearchCommandPaletteLandersRow, error)
	SearchCommandPaletteOffers(ctx context.Context, arg db.SearchCommandPaletteOffersParams) ([]db.SearchCommandPaletteOffersRow, error)
}

type Store struct {
	q paletteQuerier
}

func NewStore(pool *pgxpool.Pool) *Store {
	if pool == nil {
		return nil
	}
	return &Store{q: db.New(pool)}
}

func (st *Store) SearchEntities(ctx context.Context, customerID uuid.UUID, query string, limit int, kinds []string) ([]ItemDTO, error) {
	if st == nil || st.q == nil {
		return nil, fmt.Errorf("command palette store unavailable")
	}
	query = truncateSearchQuery(query)
	if query == "" {
		return nil, nil
	}
	if customerID == uuid.Nil {
		return nil, fmt.Errorf("customer_id is required")
	}
	if limit <= 0 {
		limit = DefaultSearchLimit
	}
	if limit > MaxSearchLimit {
		limit = MaxSearchLimit
	}
	kindSet := parseSearchKinds(kinds)
	if !kindSet.any() {
		return nil, nil
	}
	perKind := perKindSearchBudget
	if perKind > limit {
		perKind = limit
	}
	queryParam := pgText(query)
	customerParam := domain.ToUUID(customerID)
	limitParam := int32(perKind)

	var batches [][]searchCandidate
	if kindSet.searchCampaigns {
		rows, err := st.q.SearchCommandPaletteCampaigns(ctx, db.SearchCommandPaletteCampaignsParams{
			CustomerID:  customerParam,
			Query:       queryParam,
			ResultLimit: limitParam,
		})
		if err != nil {
			return nil, err
		}
		batches = append(batches, campaignRowsToCandidates(query, rows))
	}
	if kindSet.searchFlows {
		rows, err := st.q.SearchCommandPaletteFlows(ctx, db.SearchCommandPaletteFlowsParams{
			Query:       queryParam,
			CustomerID:  customerParam,
			ResultLimit: limitParam,
		})
		if err != nil {
			return nil, err
		}
		batches = append(batches, flowRowsToCandidates(query, rows))
	}
	if kindSet.searchLanders {
		rows, err := st.q.SearchCommandPaletteLanders(ctx, db.SearchCommandPaletteLandersParams{
			Query:       queryParam,
			ResultLimit: limitParam,
		})
		if err != nil {
			return nil, err
		}
		batches = append(batches, landerRowsToCandidates(query, rows))
	}
	if kindSet.searchOffers {
		rows, err := st.q.SearchCommandPaletteOffers(ctx, db.SearchCommandPaletteOffersParams{
			Query:       queryParam,
			ResultLimit: limitParam,
		})
		if err != nil {
			return nil, err
		}
		batches = append(batches, offerRowsToCandidates(query, rows))
	}
	return mergeSearchResults(limit, batches...), nil
}

type searchKindSet struct {
	searchCampaigns bool
	searchFlows     bool
	searchLanders   bool
	searchOffers    bool
}

func (s searchKindSet) any() bool {
	return s.searchCampaigns || s.searchFlows || s.searchLanders || s.searchOffers
}

func parseSearchKinds(kinds []string) searchKindSet {
	if len(kinds) == 0 {
		return searchKindSet{
			searchCampaigns: true,
			searchFlows:     true,
			searchLanders:   true,
			searchOffers:    true,
		}
	}
	var set searchKindSet
	for _, kind := range kinds {
		switch strings.ToLower(strings.TrimSpace(kind)) {
		case "campaign":
			set.searchCampaigns = true
		case "flow":
			set.searchFlows = true
		case "lander":
			set.searchLanders = true
		case "offer":
			set.searchOffers = true
		}
	}
	return set
}

func campaignRowsToCandidates(query string, rows []db.SearchCommandPaletteCampaignsRow) []searchCandidate {
	out := make([]searchCandidate, 0, len(rows))
	for _, row := range rows {
		id := uuidFromPG(row.ID)
		status := strings.ToUpper(string(row.Status))
		out = append(out, searchCandidate{
			item: ItemDTO{
				ID:          id.String(),
				Kind:        "campaign",
				Label:       row.Name,
				StatusLabel: campaignStatusLabel(status),
				StatusTone:  campaignStatusTone(status),
				Href:        "/campaigns/" + id.String(),
				Group:       "campaigns",
			},
			prefixRank: prefixRank(query, row.Name),
			sortTime:   timeFromPG(row.UpdatedAt),
		})
	}
	return out
}

func flowRowsToCandidates(query string, rows []db.SearchCommandPaletteFlowsRow) []searchCandidate {
	out := make([]searchCandidate, 0, len(rows))
	for _, row := range rows {
		id := uuidFromPG(row.ID)
		out = append(out, searchCandidate{
			item: ItemDTO{
				ID:    id.String(),
				Kind:  "flow",
				Label: row.Name,
				Href:  "/flows/" + id.String(),
				Group: "campaigns",
			},
			prefixRank: prefixRank(query, row.Name),
			sortTime:   timeFromPG(row.CreatedAt),
		})
	}
	return out
}

func landerRowsToCandidates(query string, rows []db.SearchCommandPaletteLandersRow) []searchCandidate {
	out := make([]searchCandidate, 0, len(rows))
	for _, row := range rows {
		id := uuidFromPG(row.ID)
		out = append(out, searchCandidate{
			item: ItemDTO{
				ID:    id.String(),
				Kind:  "lander",
				Label: row.Name,
				Href:  "/landers/" + id.String(),
				Group: "campaigns",
			},
			prefixRank: prefixRank(query, row.Name),
			sortTime:   timeFromPG(row.CreatedAt),
		})
	}
	return out
}

func offerRowsToCandidates(query string, rows []db.SearchCommandPaletteOffersRow) []searchCandidate {
	out := make([]searchCandidate, 0, len(rows))
	for _, row := range rows {
		id := uuidFromPG(row.ID)
		out = append(out, searchCandidate{
			item: ItemDTO{
				ID:    id.String(),
				Kind:  "offer",
				Label: row.Name,
				Href:  "/offers/" + id.String(),
				Group: "campaigns",
			},
			prefixRank: prefixRank(query, row.Name),
			sortTime:   timeFromPG(row.CreatedAt),
		})
	}
	return out
}

func pgText(value string) pgtype.Text {
	return pgtype.Text{String: value, Valid: true}
}

func uuidFromPG(id pgtype.UUID) uuid.UUID {
	if !id.Valid {
		return uuid.Nil
	}
	return uuid.UUID(id.Bytes)
}

func timeFromPG(ts pgtype.Timestamptz) time.Time {
	if !ts.Valid {
		return time.Time{}
	}
	return ts.Time
}
