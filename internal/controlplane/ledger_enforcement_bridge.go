package controlplane

import (
	"context"
	"fmt"

	"ad-event-processor/internal/automation"
	"ad-event-processor/internal/config"
	"ad-event-processor/internal/ledger"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// LedgerEnforcementHost routes margin-guard worker enforcement through campaign pause
// and placement blacklist APIs instead of raw outbox inserts.
type LedgerEnforcementHost struct {
	svc *Service
}

func NewLedgerEnforcementHost(svc *Service) ledger.EnforcementHost {
	return LedgerEnforcementHost{svc: svc}
}

func (h LedgerEnforcementHost) PauseCampaign(ctx context.Context, campaignID uuid.UUID, reason string) error {
	if h.svc == nil {
		return ledger.ErrEnforcementUnavailable()
	}
	return h.svc.PauseCampaign(ctx, campaignID, reason)
}

func (h LedgerEnforcementHost) BlacklistPlacement(ctx context.Context, campaignID uuid.UUID, placementID string) error {
	if h.svc == nil {
		return ledger.ErrEnforcementUnavailable()
	}
	return h.svc.BlockCampaignPlacement(ctx, campaignID, placementID)
}

func (h LedgerEnforcementHost) PlatformPauseCampaign(ctx context.Context, campaignID uuid.UUID, network, reason string) error {
	if h.svc == nil {
		return ledger.ErrEnforcementUnavailable()
	}
	exec := automation.NewExecutor(automationHost{svc: h.svc})
	idempotencyKey := fmt.Sprintf("margin-guard:%s:%s", campaignID, reason)
	return exec.PlatformPause(ctx, uuid.Nil, campaignID, network, idempotencyKey)
}

// NewMarginGuardEnforcementService builds a pool-only Service for the margin-guard sidecar.
// It does not start outbox or admin workers; enforcement writes Postgres state and enqueues outbox.
func NewMarginGuardEnforcementService(pool *pgxpool.Pool, cfg *config.Config) *Service {
	if cfg == nil {
		cfg = &config.Config{}
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &Service{
		pool:   pool,
		cfg:    cfg,
		ctx:    ctx,
		cancel: cancel,
	}
}
