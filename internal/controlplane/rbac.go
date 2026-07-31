package controlplane

import "strings"

const (
	RoleAdmin   = "A"
	RoleManager = "M"
	RoleUser    = "U"
	RoleBuyer   = "B"
	RoleSupport = "S"
)

var rolePermissions = map[string][]string{
	RoleAdmin: {
		"customers:write", "customers:read",
		"campaigns:write", "campaigns:read",
		"brands:write", "brands:read",
		"settings:write", "settings:read",
		"blacklist:write", "blacklist:read",
		"audit:read",
		"users:write",
		"shards:write", "shards:read",
	},
	RoleManager: {
		"customers:write", "customers:read",
		"campaigns:write", "campaigns:read",
		"brands:write", "brands:read",
		"audit:read",
	},
	RoleUser: {
		"campaigns:write", "campaigns:read",
		"customers:read",
		"brands:write", "brands:read",
	},
	RoleBuyer: {
		"campaigns:read:masked", "campaigns:pause",
	},
	RoleSupport: {
		"campaigns:read:masked", "audit:read",
	},
}

func NormalizeRole(role string) string {
	switch strings.ToUpper(strings.TrimSpace(role)) {
	case "SUPERADMIN", "ADMIN", "SA", "A":
		return RoleAdmin
	case "MANAGER", "M":
		return RoleManager
	case "CUSTOMER", "USER", "C", "U":
		return RoleUser
	case "BUYER", "B":
		return RoleBuyer
	case "SUPPORT", "S":
		return RoleSupport
	default:
		return strings.ToUpper(strings.TrimSpace(role))
	}
}

func GetPermissionsForRole(role string) []string {
	perms, exists := rolePermissions[NormalizeRole(role)]
	if !exists {
		return []string{}
	}
	return perms
}

func HasPermission(role, permission string) bool {
	for _, p := range rolePermissions[NormalizeRole(role)] {
		if p == permission {
			return true
		}
	}
	return false
}
