package ledger

import (
	"context"

	"github.com/google/uuid"
)

// EnforcementHost applies margin-guard pause decisions through campaign lifecycle and
// placement blacklist APIs. Callers must not enqueue PAUSE_CAMPAIGN or PAUSE_PLACEMENT
// outbox rows directly from the worker.
type EnforcementHost interface {
	PauseCampaign(ctx context.Context, campaignID uuid.UUID, reason string) error
	BlacklistPlacement(ctx context.Context, campaignID uuid.UUID, placementID string) error
	PlatformPauseCampaign(ctx context.Context, campaignID uuid.UUID, network, reason string) error
}
