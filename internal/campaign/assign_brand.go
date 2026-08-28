package campaign

import (
	"context"

	"ad-event-processor/internal/controlplane/authz"
	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type brandAssignAuditChange struct {
	BrandID string `json:"brand_id"`
}

func AssignCampaignBrand(ctx context.Context, fx Effects, pool *pgxpool.Pool, campaignID, brandID uuid.UUID) error {
	if fx == nil || pool == nil {
		return errServiceUnavailable()
	}
	if campaignID == uuid.Nil {
		return errValidation("campaign id required")
	}
	camp, err := fx.GetCampaignRow(ctx, campaignID)
	if err != nil {
		return mapCampaignStoreError(err)
	}
	customerID := uuid.UUID(camp.CustomerID.Bytes)

	brandFcapKey := "fcap:c:" + campaignID.String()
	brandArg := brandIDOrNil(uuid.Nil)
	auditBrandID := ""
	if brandID != uuid.Nil {
		q := db.New(pool)
		brand, err := q.GetBrand(ctx, domain.ToUUID(brandID))
		if err != nil {
			return mapCampaignNotFound(err, ErrBrandNotFound)
		}
		if uuid.UUID(brand.CustomerID.Bytes) != customerID {
			return ErrBrandBelongsToAnotherCustomer
		}
		brandFcapKey = "fcap:b:" + brandID.String()
		brandArg = brandID
		auditBrandID = brandID.String()
	}

	tag, err := pool.Exec(ctx,
		`UPDATE campaigns SET brand_id = $2, brand_fcap_key = $3, updated_at = now() WHERE id = $1 AND deleted_at IS NULL`,
		campaignID, brandArg, brandFcapKey,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrCampaignNotFound
	}

	adminID := uuid.Nil
	if u, ok := authz.GetUser(ctx); ok {
		adminID = u.UserID
	}
	fx.AuditLog(ctx, db.New(pool), adminID, "PATCH_CAMPAIGN", "campaign", &campaignID, brandAssignAuditChange{
		BrandID: auditBrandID,
	}, nil)

	fx.PublishCampaignUpdate(ctx, campaignID.String())
	return nil
}

func brandIDOrNil(id uuid.UUID) any {
	if id == uuid.Nil {
		return nil
	}
	return id
}
