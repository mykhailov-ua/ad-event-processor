package track

import (
	"context"

	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
)

type Host interface {
	CampaignRegistry() domain.CampaignRegistry
	BrandCreativeStore() BrandStore
	LinkSigningSecret() []byte
}

type BrandStore interface {
	SelectLandingURLBytes(ctx context.Context, brandID uuid.UUID, userID string, evt *domain.Event) []byte
}

type TrackFilterHost interface {
	CampaignRegistry() domain.CampaignRegistry
	BrandCreativeStore() BrandStore
}

type LandingDeps interface {
	CampaignRegistry() domain.CampaignRegistry
	LinkSigningSecret() []byte
}
