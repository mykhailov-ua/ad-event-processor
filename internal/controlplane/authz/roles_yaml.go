package authz

import (
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type rolesFile struct {
	Version int                    `yaml:"version"`
	Roles   map[string]roleEntry   `yaml:"roles"`
}

type roleEntry struct {
	Scope        string   `yaml:"scope"`
	Permissions  []string `yaml:"permissions"`
	Capabilities []string `yaml:"capabilities"`
}

var rolesYAMLLoader func(path string, store *Store) error

func SetRolesYAMLLoader(loader func(path string, store *Store) error) {
	rolesYAMLLoader = loader
}

func LoadRolesYAML(path string, store *Store) error {
	if store == nil {
		return nil
	}
	if rolesYAMLLoader != nil {
		return rolesYAMLLoader(path, store)
	}
	return loadRolesYAMLLegacy(path, store)
}

func loadRolesYAMLLegacy(path string, store *Store) error {
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
		perms := entry.Permissions
		store.SetRole(strings.ToUpper(strings.TrimSpace(role)), scope, perms)
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
