package access

import (
	"fmt"
	"regexp"
	"strings"
)

var customRoleCodeRe = regexp.MustCompile(`^[A-Z][A-Z0-9_]{0,15}$`)

var builtinRoleCodes = map[string]struct{}{
	"A": {}, "M": {}, "U": {}, "B": {}, "TL": {}, "MB": {}, "S": {}, "P": {},
}

func ValidateDocument(doc RolesDocument, allowWildcard bool) ([]ValidationDetail, map[string][]string) {
	var errs []ValidationDetail
	if doc.Version != 0 && doc.Version != SchemaVersion {
		errs = append(errs, ValidationDetail{Field: "version", Code: "unsupported_version"})
	}
	if len(doc.Roles) > MaxRoles {
		errs = append(errs, ValidationDetail{Field: "roles", Code: "too_many_roles"})
	}
	knownCaps := KnownCapabilities()
	knownPerms := KnownPermissions()
	compiled := make(map[string][]string, len(doc.Roles))
	for code, role := range doc.Roles {
		normalized := normalizeRoleCode(code)
		if normalized == "" {
			errs = append(errs, ValidationDetail{Field: "roles." + code, Code: "invalid_role_code"})
			continue
		}
		if normalized != strings.TrimSpace(code) {
			errs = append(errs, ValidationDetail{Field: "roles." + code, Code: "role_code_not_uppercase"})
		}
		if !customRoleCodeRe.MatchString(normalized) {
			errs = append(errs, ValidationDetail{Field: "roles." + normalized, Code: "invalid_role_code"})
		}
		scope := strings.ToLower(strings.TrimSpace(role.Scope))
		switch scope {
		case "global", "customer", "team":
		default:
			errs = append(errs, ValidationDetail{Field: "roles." + normalized + ".scope", Code: "invalid_scope"})
		}
		for _, capID := range role.Capabilities {
			if _, ok := knownCaps[capID]; !ok {
				errs = append(errs, ValidationDetail{Field: "roles." + normalized + ".capabilities", Code: "unknown_capability"})
			}
		}
		for _, perm := range role.Permissions {
			perm = strings.TrimSpace(perm)
			if perm == "" {
				continue
			}
			if perm == "*" {
				if !allowWildcard {
					errs = append(errs, ValidationDetail{Field: "roles." + normalized + ".permissions", Code: "wildcard_forbidden"})
				}
				continue
			}
			if _, ok := knownPerms[perm]; !ok {
				errs = append(errs, ValidationDetail{Field: "roles." + normalized + ".permissions", Code: "unknown_permission"})
			}
		}
		perms := CompileRolePermissions(role)
		if len(perms) > MaxPermsPerRole {
			errs = append(errs, ValidationDetail{Field: "roles." + normalized + ".permissions", Code: "too_many_permissions"})
		}
		if len(perms) == 0 {
			errs = append(errs, ValidationDetail{Field: "roles." + normalized, Code: "empty_role"})
		}
		for _, p := range perms {
			if p == "*" && !allowWildcard {
				errs = append(errs, ValidationDetail{Field: "roles." + normalized + ".permissions", Code: "wildcard_forbidden"})
			} else if p != "*" {
				if _, ok := knownPerms[p]; !ok {
					errs = append(errs, ValidationDetail{Field: "roles." + normalized + ".permissions", Code: "unknown_permission"})
				}
			}
		}
		compiled[normalized] = perms
	}
	return errs, compiled
}

func normalizeRoleCode(code string) string {
	return strings.ToUpper(strings.TrimSpace(code))
}

func IsBuiltinRole(code string) bool {
	_, ok := builtinRoleCodes[normalizeRoleCode(code)]
	return ok
}

func BuiltinRoleCodes() []string {
	out := make([]string, 0, len(builtinRoleCodes))
	for code := range builtinRoleCodes {
		out = append(out, code)
	}
	return out
}

func ValidateAssignableTeamRole(doc RolesDocument, role string) (string, error) {
	code := normalizeRoleCode(role)
	if code == "" {
		return "", fmt.Errorf("role required")
	}
	if code == ReservedRoleCode {
		return "", fmt.Errorf("admin role assignment not allowed via team API")
	}
	roleDef, ok := doc.Roles[code]
	if !ok {
		return "", fmt.Errorf("unknown role %s", code)
	}
	scope := strings.ToLower(strings.TrimSpace(roleDef.Scope))
	if scope != "team" {
		return "", fmt.Errorf("role scope must be team")
	}
	return code, nil
}

func RolesInUseConflict(removed []string) error {
	if len(removed) == 0 {
		return nil
	}
	return fmt.Errorf("role_in_use: %s", strings.Join(removed, ","))
}
