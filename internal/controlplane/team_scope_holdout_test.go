package controlplane

import (
	"context"
	"testing"

	"ad-event-processor/internal/controlplane/authz"
	"ad-event-processor/internal/teamscope"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestTeamScope_mediaBuyerListScope_holdout(t *testing.T) {
	userID := uuid.New()
	ctx := authz.WithAuthenticatedUser(context.Background(), authz.AuthenticatedUser{
		UserID: userID,
		Role:   authz.RoleMediaBuyer,
	})
	scope, err := teamscope.ResolveListScope(ctx, nil, uuid.New())
	require.NoError(t, err)
	require.True(t, scope.OwnerUserID.Valid)
	require.Equal(t, userID, uuid.UUID(scope.OwnerUserID.Bytes))
}

func TestTeamScope_maskedTeamBlocksOwnerQueryOverride_holdout(t *testing.T) {
	ctx := authz.WithAuthenticatedUser(context.Background(), authz.AuthenticatedUser{
		UserID: uuid.New(),
		Role:   authz.RoleMediaBuyer,
	})
	ctx = authz.WithSnapshot(ctx, authz.Snapshot{
		Scope: authz.ScopeTeam,
		Mask:  authz.MaskMasked,
		Permissions: map[string]struct{}{
			authz.PermCampaignsReadMasked: {},
		},
	})
	require.False(t, teamscope.AllowOwnerQueryOverride(ctx))
}

func TestTeamScope_teamLeadWithWriteAllowsOwnerOverride(t *testing.T) {
	ctx := authz.WithAuthenticatedUser(context.Background(), authz.AuthenticatedUser{
		UserID: uuid.New(),
		Role:   authz.RoleTeamLead,
	})
	ctx = authz.WithSnapshot(ctx, authz.Snapshot{
		Scope: authz.ScopeTeam,
		Mask:  authz.MaskMasked,
		Permissions: map[string]struct{}{
			authz.PermCampaignsWrite: {},
		},
	})
	require.True(t, teamscope.AllowOwnerQueryOverride(ctx))
}
