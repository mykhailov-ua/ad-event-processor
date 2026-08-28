package controlplane

import (
	ctrlhttp "ad-event-processor/internal/control/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeRole(t *testing.T) {
	tests := []struct {
		in  string
		out string
	}{
		{"A", ctrlhttp.RoleAdmin},
		{"admin", ctrlhttp.RoleAdmin},
		{"SA", ctrlhttp.RoleAdmin},
		{"M", ctrlhttp.RoleManager},
		{"manager", ctrlhttp.RoleManager},
		{"U", ctrlhttp.RoleUser},
		{"user", ctrlhttp.RoleUser},
		{"C", ctrlhttp.RoleUser},
		{"customer", ctrlhttp.RoleUser},
		{"unknown", "UNKNOWN"},
	}
	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			assert.Equal(t, tc.out, ctrlhttp.NormalizeRole(tc.in))
		})
	}
}

func TestGetPermissionsForRole(t *testing.T) {
	adminPerms := []string{
		"*",
		"customers:write", "customers:read",
		"campaigns:write", "campaigns:read",
		"brands:write", "brands:read",
		"settings:write", "settings:read",
		"blacklist:write", "blacklist:read",
		"audit:read", "users:write",
		"shards:write", "shards:read",
		"rtb:write", "rtb:read",
	}
	managerPerms := []string{
		"customers:write", "customers:read",
		"campaigns:write", "campaigns:read",
		"brands:write", "brands:read",
		"audit:read",
	}
	userPerms := []string{
		"campaigns:write", "campaigns:read",
		"customers:read",
		"brands:write", "brands:read",
	}

	tests := []struct {
		role          string
		expectedPerms []string
	}{
		{ctrlhttp.RoleAdmin, adminPerms},
		{"admin", adminPerms},
		{"SA", adminPerms},
		{ctrlhttp.RoleManager, managerPerms},
		{"manager", managerPerms},
		{ctrlhttp.RoleUser, userPerms},
		{"user", userPerms},
		{"C", userPerms},
		{"unknown", []string{}},
	}

	for _, tc := range tests {
		t.Run(tc.role, func(t *testing.T) {
			perms := ctrlhttp.GetPermissionsForRole(tc.role)
			assert.ElementsMatch(t, tc.expectedPerms, perms)
		})
	}
}

func TestHasPermission(t *testing.T) {
	assert.True(t, ctrlhttp.HasPermission(ctrlhttp.RoleAdmin, "users:write"))
	assert.False(t, ctrlhttp.HasPermission(ctrlhttp.RoleUser, "users:write"))
	assert.True(t, ctrlhttp.HasPermission(ctrlhttp.RoleUser, "brands:write"))
}
