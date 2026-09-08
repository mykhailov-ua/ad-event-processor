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

	dto, ok := resolveBootstrapAuthUser(ctx, nil)
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

	dto, ok := resolveBootstrapAuthUser(ctx, nil)
	require.True(t, ok)
	require.False(t, snap.Has(ctrlhttp.PermShardsRead))
	require.NotContains(t, dto.Permissions, ctrlhttp.PermShardsRead)
}

func TestResolveBootstrapPermissions_policyStoreNoSnapshot_holdout(t *testing.T) {
	store := ctrlhttp.InitPolicyStore()
	authMW := NewAuthMiddleware(nil, nil, nil, nil)
	authMW.SetPolicyStore(store)

	u := authz.AuthenticatedUser{
		UserID:     uuid.New(),
		Role:       ctrlhttp.RoleAdmin,
		CustomerID: uuid.New(),
	}
	ctx := authz.WithAuthenticatedUser(context.Background(), u)

	dto, ok := resolveBootstrapAuthUser(ctx, authMW)
	require.True(t, ok)
	expected := authz.PermissionsList(store.EffectivePermissions(u.UserID, u.Role))
	require.Equal(t, expected, dto.Permissions)
	require.NotContains(t, dto.Permissions, ctrlhttp.PermRtbRead)
	require.Contains(t, ctrlhttp.GetPermissionsForRole(ctrlhttp.RoleAdmin), ctrlhttp.PermRtbRead)
}
