package access

import (
	"context"
	"time"
)

const (
	SchemaVersion    = 1
	MaxRoles         = 128
	MaxPermsPerRole  = 64
	MaxFileBytes     = 256 * 1024
	ReservedRoleCode = "A"
)

type RolesDocument struct {
	Version   int                       `json:"version" yaml:"version"`
	Revision  int                       `json:"revision" yaml:"revision"`
	UpdatedAt string                    `json:"updated_at,omitempty" yaml:"updated_at,omitempty"`
	UpdatedBy string                    `json:"updated_by,omitempty" yaml:"updated_by,omitempty"`
	Roles     map[string]RoleDefinition `json:"roles" yaml:"roles"`
}

type RoleDefinition struct {
	Scope        string   `json:"scope" yaml:"scope"`
	Builtin      bool     `json:"builtin,omitempty" yaml:"builtin,omitempty"`
	Label        string   `json:"label,omitempty" yaml:"label,omitempty"`
	Capabilities []string `json:"capabilities,omitempty" yaml:"capabilities,omitempty"`
	Permissions  []string `json:"permissions,omitempty" yaml:"permissions,omitempty"`
}

type RoleView struct {
	Code                string   `json:"code"`
	Scope               string   `json:"scope"`
	Builtin             bool     `json:"builtin"`
	Label               string   `json:"label,omitempty"`
	Capabilities        []string `json:"capabilities,omitempty"`
	Permissions         []string `json:"permissions,omitempty"`
	CompiledPermissions []string `json:"compiled_permissions"`
	MemberCount         int64    `json:"member_count,omitempty"`
}

type CatalogResponse struct {
	Capabilities []CapabilityEntry `json:"capabilities"`
	Permissions  []PermissionEntry `json:"permissions"`
}

type CapabilityEntry struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Group       string `json:"group"`
	Description string `json:"description,omitempty"`
}

type PermissionEntry struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Internal    bool   `json:"internal,omitempty"`
	Description string `json:"description,omitempty"`
}

type ValidationDetail struct {
	Field string `json:"field"`
	Code  string `json:"code"`
}

type ApplyResult struct {
	Revision int                 `json:"revision"`
	Roles    map[string]RoleView `json:"roles"`
}

type ValidateResponse struct {
	Valid  bool                `json:"valid"`
	Errors []ValidationDetail  `json:"errors,omitempty"`
	Roles  map[string]RoleView `json:"roles,omitempty"`
}

type ApplyOptions struct {
	ExpectedRevision int
	ActorUserID      string
	AllowWildcard    bool
	Audit            AuditWriter
}

type AuditWriter func(ctx context.Context, action string, changes, metadata any)

func nowUTC() string {
	return time.Now().UTC().Format(time.RFC3339)
}
