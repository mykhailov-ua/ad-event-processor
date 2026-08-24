package ingestion

import (
	"context"
	"time"

	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/pkg/piihash"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	redis "github.com/redis/go-redis/v9"
)

const conversionEventType = "conversion"

type segmentCampaignLoader interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Campaign, error)
}

type SegmentConversionHandler struct {
	repo    segmentCampaignLoader
	queries db.Querier
	rdbs    []redis.UniversalClient
	hasher  *piihash.Hasher
}

func NewSegmentConversionHandler(repo segmentCampaignLoader, queries db.Querier, rdbs []redis.UniversalClient, hasher *piihash.Hasher) *SegmentConversionHandler {
	return &SegmentConversionHandler{
		repo:    repo,
		queries: queries,
		rdbs:    rdbs,
		hasher:  hasher,
	}
}

func (h *SegmentConversionHandler) Handle(evt *domain.Event, _ string) {
	if h == nil || evt == nil || evt.Type != conversionEventType || evt.UserID == "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	camp, err := h.repo.GetByID(ctx, evt.CampaignID)
	if err != nil || camp == nil || camp.RetargetSegmentID == uuid.Nil {
		return
	}
	ttlHours := camp.SegmentTTLHours
	if ttlHours <= 0 {
		return
	}
	userHash, ok := segmentUserHash(h.hasher, evt)
	if !ok {
		return
	}
	ttl := time.Duration(ttlHours) * time.Hour
	if err := addSegmentMember(ctx, h.rdbs, camp.RetargetSegmentID, userHash, ttl); err != nil {
		return
	}
	if h.queries == nil {
		return
	}
	expiresAt := pgtype.Timestamptz{Time: time.Now().Add(ttl), Valid: true}
	_ = h.queries.UpsertSegmentMember(ctx, db.UpsertSegmentMemberParams{
		SegmentID: pgtype.UUID{Bytes: camp.RetargetSegmentID, Valid: true},
		UserHash:  userHash[:],
		ExpiresAt: expiresAt,
	})
}
