package campaign

import (
	"context"

	"ad-event-processor/internal/controlplane/authz"
	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func EnforcePublishGate(ctx context.Context, fx Effects, pool *pgxpool.Pool, campaignID uuid.UUID, row db.Campaign, force bool) error {
	if force {
		return AuditPublishForce(ctx, fx, pool, campaignID)
	}
	blocked, err := CollectPublishBlocked(ctx, fx, campaignID, row)
	if err != nil {
		return err
	}
	if blocked != nil {
		return blocked
	}
	return nil
}

func AuditPublishForce(ctx context.Context, fx Effects, pool *pgxpool.Pool, campaignID uuid.UUID) error {
	if fx == nil || pool == nil {
		return errServiceUnavailable()
	}
	return pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		q := db.New(tx)
		var uid uuid.UUID
		if user, ok := authz.GetUser(ctx); ok {
			uid = user.UserID
		}
		fx.AuditLog(ctx, q, uid, "PUBLISH_FORCE", "campaign", &campaignID, auditReasonChange{Reason: "publish_gate_bypass"}, nil)
		return nil
	})
}
