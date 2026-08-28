package campaign

import (
	"context"

	"ad-event-processor/internal/controlplane/authz"
	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func CampaignOwnerUserFilter(ctx context.Context) pgtype.UUID {
	u, ok := authz.GetUser(ctx)
	if !ok || u.UserID == uuid.Nil {
		return pgtype.UUID{}
	}
	if authz.NormalizeRole(u.Role) == authz.RoleMediaBuyer {
		return domain.ToUUID(u.UserID)
	}
	return pgtype.UUID{}
}

func mediaBuyerOwnsCampaign(u authz.AuthenticatedUser, camp db.Campaign) bool {
	if authz.NormalizeRole(u.Role) != authz.RoleMediaBuyer {
		return true
	}
	if !camp.OwnerUserID.Valid {
		return false
	}
	return uuid.UUID(camp.OwnerUserID.Bytes) == u.UserID
}

func AssertMediaBuyerCampaignAccess(ctx context.Context, camp db.Campaign) error {
	u, ok := authz.GetUser(ctx)
	if !ok {
		return nil
	}
	if !mediaBuyerOwnsCampaign(u, camp) {
		return ErrForbidden
	}
	return nil
}

func campaignOwnerUserFilter(ctx context.Context) pgtype.UUID {
	return CampaignOwnerUserFilter(ctx)
}

func assertMediaBuyerCampaignAccess(ctx context.Context, camp db.Campaign) error {
	return AssertMediaBuyerCampaignAccess(ctx, camp)
}
