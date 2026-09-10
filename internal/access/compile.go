package access

import (
	"sort"
	"strings"
)

func CompileRolePermissions(role RoleDefinition) []string {
	set := make(map[string]struct{})
	for _, capID := range role.Capabilities {
		for _, p := range capabilityPermissions[capID] {
			set[p] = struct{}{}
		}
	}
	for _, p := range role.Permissions {
		p = strings.TrimSpace(p)
		if p != "" {
			set[p] = struct{}{}
		}
	}
	if len(set) == 0 {
		return nil
	}
	out := make([]string, 0, len(set))
	for p := range set {
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}

func CompileDocument(doc RolesDocument) map[string][]string {
	out := make(map[string][]string, len(doc.Roles))
	for code, role := range doc.Roles {
		out[code] = CompileRolePermissions(role)
	}
	return out
}
