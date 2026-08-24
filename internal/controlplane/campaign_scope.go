package controlplane

import (
	"context"

	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func campaignOwnerUserFilter(ctx context.Context) pgtype.UUID {
	u, ok := GetUser(ctx)
	if !ok || u.UserID == uuid.Nil {
		return pgtype.UUID{}
	}
	if NormalizeRole(u.Role) == RoleMediaBuyer {
		return domain.ToUUID(u.UserID)
	}
	return pgtype.UUID{}
}

func mediaBuyerOwnsCampaign(u AuthenticatedUser, camp db.Campaign) bool {
	if NormalizeRole(u.Role) != RoleMediaBuyer {
		return true
	}
	if !camp.OwnerUserID.Valid {
		return false
	}
	return uuid.UUID(camp.OwnerUserID.Bytes) == u.UserID
}

func assertMediaBuyerCampaignAccess(ctx context.Context, camp db.Campaign) error {
	u, ok := GetUser(ctx)
	if !ok {
		return nil
	}
	if !mediaBuyerOwnsCampaign(u, camp) {
		return errForbidden
	}
	return nil
}
