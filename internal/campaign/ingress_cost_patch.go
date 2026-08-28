package campaign

import (
	"context"

	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ingressCostPatchHost interface {
	PublishCampaignUpdate(ctx context.Context, campaignID string)
}

func ApplyIngressCostPatch(
	ctx context.Context,
	host ingressCostPatchHost,
	pool *pgxpool.Pool,
	campaignID uuid.UUID,
	cfg IngressCostConfigDTO,
) error {
	if pool == nil || host == nil {
		return errServiceUnavailable()
	}
	err := pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		return applyCampaignIngressCostPatchTx(ctx, db.New(tx), campaignID, cfg)
	})
	if err != nil {
		return err
	}
	host.PublishCampaignUpdate(ctx, campaignID.String())
	return nil
}
