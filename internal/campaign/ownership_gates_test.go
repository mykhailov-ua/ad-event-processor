package campaign

import (
	"context"
	"testing"

	"ad-event-processor/internal/controlplane/authz"
	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestCampaignOwnerUserFilter_mediaBuyerScoped(t *testing.T) {
	t.Parallel()
	ctx := authz.WithAuthenticatedUser(context.Background(), authz.AuthenticatedUser{
		Role:   authz.RoleMediaBuyer,
		UserID: uuid.MustParse("11111111-1111-1111-1111-111111111111"),
	})
	filter := campaignOwnerUserFilter(ctx)
	assert.True(t, filter.Valid)
}

func TestCampaignOwnerUserFilter_adminUnscoped(t *testing.T) {
	t.Parallel()
	ctx := authz.WithAuthenticatedUser(context.Background(), authz.AuthenticatedUser{
		Role:   authz.RoleAdmin,
		UserID: uuid.New(),
	})
	filter := campaignOwnerUserFilter(ctx)
	assert.False(t, filter.Valid)
}

func TestAssertMediaBuyerCampaignAccess_forbidden_holdout(t *testing.T) {
	t.Parallel()
	ownerID := uuid.New()
	otherBuyer := uuid.New()
	ctx := authz.WithAuthenticatedUser(context.Background(), authz.AuthenticatedUser{
		Role:   authz.RoleMediaBuyer,
		UserID: otherBuyer,
	})
	camp := db.Campaign{
		OwnerUserID: domain.ToUUID(ownerID),
	}
	assert.ErrorIs(t, assertMediaBuyerCampaignAccess(ctx, camp), ErrForbidden)
}
