package controlplane

import (
	"context"
	"testing"

	ctrlhttp "ad-event-processor/internal/control/http"
	"ad-event-processor/internal/controlplane/authz"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestResolveBootstrapPermissions_matchSnapshot_holdout(t *testing.T) {
	store := ctrlhttp.InitPolicyStore()
	snap := store.EffectivePermissions(uuid.Nil, ctrlhttp.RoleAdmin)
	ctx := authz.WithSnapshot(context.Background(), snap)

	u := authz.AuthenticatedUser{
		UserID:     uuid.New(),
		Role:       ctrlhttp.RoleAdmin,
		CustomerID: uuid.New(),
	}
	ctx = authz.WithAuthenticatedUser(ctx, u)

	dto, ok := resolveBootstrapAuthUser(ctx)
	require.True(t, ok)
	require.Equal(t, authz.PermissionsList(snap), dto.Permissions)
	require.NotContains(t, dto.Permissions, ctrlhttp.PermRtbRead)
	require.Contains(t, ctrlhttp.GetPermissionsForRole(ctrlhttp.RoleAdmin), ctrlhttp.PermRtbRead)
}

func TestResolveBootstrapPermissions_mediaBuyerDeniesOps_holdout(t *testing.T) {
	store := ctrlhttp.InitPolicyStore()
	snap := store.EffectivePermissions(uuid.Nil, ctrlhttp.RoleMediaBuyer)
	ctx := authz.WithSnapshot(context.Background(), snap)

	u := authz.AuthenticatedUser{
		UserID:     uuid.New(),
		Role:       ctrlhttp.RoleMediaBuyer,
		CustomerID: uuid.New(),
	}
	ctx = authz.WithAuthenticatedUser(ctx, u)

	dto, ok := resolveBootstrapAuthUser(ctx)
	require.True(t, ok)
	require.False(t, snap.Has(ctrlhttp.PermShardsRead))
	require.NotContains(t, dto.Permissions, ctrlhttp.PermShardsRead)
}
