package authz_test

import (
	"testing"

	"github.com/bidshard/ad-event-processor/internal/controlplane/authz"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMaskLevelFromPermissions(t *testing.T) {
	full := map[string]struct{}{authz.PermCampaignsRead: {}}
	masked := map[string]struct{}{authz.PermCampaignsReadMasked: {}}
	assert.Equal(t, authz.MaskFull, authz.MaskLevelFromPermissions(full))
	assert.Equal(t, authz.MaskMasked, authz.MaskLevelFromPermissions(masked))
}

func TestPolicyPermissionMatrix(t *testing.T) {
	store := authz.NewStore()
	store.SetRole("A", authz.ScopeGlobal, []string{"campaigns:read", "campaigns:write", "audit:read"})
	store.SetRole("B", authz.ScopeTeam, []string{"campaigns:read:masked", "campaigns:pause"})
	store.SetRole("S", authz.ScopeGlobal, []string{"campaigns:read:masked", "audit:read"})
	store.SetRole("M", authz.ScopeCustomer, []string{"campaigns:read", "campaigns:write"})

	cases := []struct {
		name      string
		role      string
		perm      string
		want      bool
		wantMask  authz.MaskLevel
		wantScope authz.Scope
	}{
		{"admin_read_campaigns", "A", authz.PermCampaignsRead, true, authz.MaskFull, authz.ScopeGlobal},
		{"admin_pause", "A", authz.PermCampaignsPause, false, authz.MaskFull, authz.ScopeGlobal},
		{"buyer_read_masked", "B", authz.PermCampaignsReadMasked, true, authz.MaskMasked, authz.ScopeTeam},
		{"buyer_full_read_denied", "B", authz.PermCampaignsRead, false, authz.MaskMasked, authz.ScopeTeam},
		{"buyer_pause", "B", authz.PermCampaignsPause, true, authz.MaskMasked, authz.ScopeTeam},
		{"buyer_write_denied", "B", authz.PermCampaignsWrite, false, authz.MaskMasked, authz.ScopeTeam},
		{"buyer_audit_denied", "B", "audit:read", false, authz.MaskMasked, authz.ScopeTeam},
		{"support_masked_read", "S", authz.PermCampaignsReadMasked, true, authz.MaskMasked, authz.ScopeGlobal},
		{"support_audit", "S", "audit:read", true, authz.MaskMasked, authz.ScopeGlobal},
		{"support_write_denied", "S", authz.PermCampaignsWrite, false, authz.MaskMasked, authz.ScopeGlobal},
		{"manager_full_read", "M", authz.PermCampaignsRead, true, authz.MaskFull, authz.ScopeCustomer},
		{"manager_pause_denied", "M", authz.PermCampaignsPause, false, authz.MaskFull, authz.ScopeCustomer},
		{"manager_audit_denied", "M", "audit:read", false, authz.MaskFull, authz.ScopeCustomer},
		{"unknown_role_denied", "Z", authz.PermCampaignsRead, false, authz.MaskMasked, authz.ScopeCustomer},
		{"buyer_write_masked_denied", "B", authz.PermCampaignsWriteMask, false, authz.MaskMasked, authz.ScopeTeam},
		{"admin_audit", "A", "audit:read", true, authz.MaskFull, authz.ScopeGlobal},
		{"admin_settings_denied_without_perm", "A", "settings:read", false, authz.MaskFull, authz.ScopeGlobal},
		{"buyer_customers_denied", "B", "customers:read", false, authz.MaskMasked, authz.ScopeTeam},
		{"manager_write", "M", authz.PermCampaignsWrite, true, authz.MaskFull, authz.ScopeCustomer},
		{"support_full_read_denied", "S", authz.PermCampaignsRead, false, authz.MaskMasked, authz.ScopeGlobal},
	}

	require.Len(t, cases, 20)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			snap := store.EffectivePermissions(uuid.Nil, tc.role)
			assert.Equal(t, tc.want, snap.Has(tc.perm), "permission %s", tc.perm)
			assert.Equal(t, tc.wantMask, snap.Mask)
			assert.Equal(t, tc.wantScope, snap.Scope)
		})
	}
}
