package authz

import (
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type rolesFile struct {
	Roles map[string]roleEntry `yaml:"roles"`
}

type roleEntry struct {
	Scope       string   `yaml:"scope"`
	Permissions []string `yaml:"permissions"`
}

func LoadRolesYAML(path string, store *Store) error {
	if store == nil {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var doc rolesFile
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return err
	}
	for role, entry := range doc.Roles {
		scope := Scope(strings.ToLower(strings.TrimSpace(entry.Scope)))
		switch scope {
		case ScopeGlobal, ScopeCustomer, ScopeTeam:
		default:
			scope = ScopeCustomer
		}
		store.SetRole(strings.ToUpper(strings.TrimSpace(role)), scope, entry.Permissions)
	}
	store.Reload()
	return nil
}

func DefaultRolesPath() string {
	if p := os.Getenv("OPERATOR_ROLES_YAML"); p != "" {
		return p
	}
	for _, candidate := range []string{"deploy/operator/roles.yaml", "../deploy/operator/roles.yaml", "../../deploy/operator/roles.yaml"} {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return "deploy/operator/roles.yaml"
}
