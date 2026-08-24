package controlplane

import (
	"log/slog"

	"ad-event-processor/internal/controlplane/authz"
)

func InitPolicyStore() *authz.Store {
	store := authz.NewStore()
	for role, perms := range rolePermissions {
		store.SetRole(role, roleScopeDefault(role), perms)
	}
	if err := authz.LoadRolesYAML(authz.DefaultRolesPath(), store); err != nil {
		slog.Warn("operator roles yaml not loaded", "path", authz.DefaultRolesPath(), "error", err)
	}
	return store
}

func roleScopeDefault(role string) authz.Scope {
	switch NormalizeRole(role) {
	case RoleAdmin, RoleSupport:
		return authz.ScopeGlobal
	case RoleBuyer, RoleTeamLead, RoleMediaBuyer:
		return authz.ScopeTeam
	default:
		return authz.ScopeCustomer
	}
}
