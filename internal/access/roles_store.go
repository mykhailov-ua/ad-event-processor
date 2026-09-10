package access

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"ad-event-processor/internal/controlplane/authz"

	"gopkg.in/yaml.v3"
)

type RolesStore struct {
	mu     sync.RWMutex
	path   string
	policy *authz.Store
	doc    RolesDocument
}

func NewRolesStore(path string, policy *authz.Store) *RolesStore {
	return &RolesStore{path: path, policy: policy}
}

func (st *RolesStore) Path() string {
	if st == nil {
		return ""
	}
	return st.path
}

func (st *RolesStore) Current() RolesDocument {
	if st == nil {
		return RolesDocument{}
	}
	st.mu.RLock()
	defer st.mu.RUnlock()
	return st.doc
}

func (st *RolesStore) Revision() int {
	return st.Current().Revision
}

func (st *RolesStore) LoadFromDisk() error {
	if st == nil {
		return errors.New("roles store unavailable")
	}
	data, err := os.ReadFile(st.path)
	if err != nil {
		return err
	}
	if len(data) > MaxFileBytes {
		return fmt.Errorf("roles file exceeds %d bytes", MaxFileBytes)
	}
	doc, err := ParseYAML(data)
	if err != nil {
		return err
	}
	return st.applyParsed(doc, false)
}

func ParseYAML(data []byte) (RolesDocument, error) {
	var raw rolesYAML
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return RolesDocument{}, err
	}
	doc := raw.toDocument()
	if doc.Version == 0 {
		doc.Version = SchemaVersion
	}
	return doc, nil
}

func (st *RolesStore) ValidateBytes(data []byte, allowWildcard bool) (ValidateResponse, error) {
	if len(data) > MaxFileBytes {
		return ValidateResponse{Valid: false, Errors: []ValidationDetail{{Field: "file", Code: "file_too_large"}}}, nil
	}
	doc, err := ParseYAML(data)
	if err != nil {
		return ValidateResponse{Valid: false, Errors: []ValidationDetail{{Field: "yaml", Code: "parse_error"}}}, nil
	}
	errs, compiled := ValidateDocument(doc, allowWildcard)
	if len(errs) > 0 {
		return ValidateResponse{Valid: false, Errors: errs}, nil
	}
	return ValidateResponse{Valid: true, Roles: viewsFromDoc(doc, compiled, nil)}, nil
}

func (st *RolesStore) ApplyBytes(ctx context.Context, data []byte, opts ApplyOptions) (ApplyResult, error) {
	if st == nil {
		return ApplyResult{}, errors.New("roles store unavailable")
	}
	if len(data) > MaxFileBytes {
		return ApplyResult{}, fmt.Errorf("roles file exceeds %d bytes", MaxFileBytes)
	}
	doc, err := ParseYAML(data)
	if err != nil {
		return ApplyResult{}, err
	}
	errs, compiled := ValidateDocument(doc, opts.AllowWildcard)
	if len(errs) > 0 {
		return ApplyResult{}, validationError(errs)
	}
	st.mu.RLock()
	current := st.doc
	st.mu.RUnlock()
	if opts.ExpectedRevision > 0 && current.Revision > 0 && opts.ExpectedRevision != current.Revision {
		return ApplyResult{}, errRevisionConflict
	}
	if err := st.ensureRolesNotRemoved(ctx, current, doc); err != nil {
		return ApplyResult{}, err
	}
	if doc.Revision <= current.Revision {
		doc.Revision = current.Revision + 1
	}
	if doc.Version == 0 {
		doc.Version = SchemaVersion
	}
	doc.UpdatedAt = nowUTC()
	doc.UpdatedBy = strings.TrimSpace(opts.ActorUserID)
	if err := st.writeAtomic(data, doc); err != nil {
		return ApplyResult{}, err
	}
	if err := st.applyParsed(doc, true); err != nil {
		return ApplyResult{}, err
	}
	if opts.Audit != nil {
		opts.Audit("access_roles_apply", map[string]any{
			"revision": doc.Revision,
			"path":     st.path,
		}, map[string]any{"roles": len(doc.Roles)})
	}
	return ApplyResult{Revision: doc.Revision, Roles: viewsFromDoc(doc, compiled, nil)}, nil
}

func (st *RolesStore) ApplyDocument(ctx context.Context, doc RolesDocument, opts ApplyOptions) (ApplyResult, error) {
	data, err := MarshalYAML(doc)
	if err != nil {
		return ApplyResult{}, err
	}
	return st.ApplyBytes(ctx, data, opts)
}

func (st *RolesStore) Reload() error {
	return st.LoadFromDisk()
}

func (st *RolesStore) RawYAML() ([]byte, error) {
	st.mu.RLock()
	path := st.path
	st.mu.RUnlock()
	return os.ReadFile(path)
}

func (st *RolesStore) applyParsed(doc RolesDocument, bumpRevision bool) error {
	errs, compiled := ValidateDocument(doc, true)
	if len(errs) > 0 {
		return validationError(errs)
	}
	if st.policy != nil {
		for code, role := range doc.Roles {
			scope := authz.Scope(strings.ToLower(strings.TrimSpace(role.Scope)))
			switch scope {
			case authz.ScopeGlobal, authz.ScopeCustomer, authz.ScopeTeam:
			default:
				scope = authz.ScopeCustomer
			}
			st.policy.SetRole(normalizeRoleCode(code), scope, compiled[normalizeRoleCode(code)])
		}
		st.policy.Reload()
	}
	st.mu.Lock()
	if bumpRevision || st.doc.Revision == 0 {
		st.doc = doc
	} else {
		st.doc = doc
	}
	st.mu.Unlock()
	return nil
}

func (st *RolesStore) ensureRolesNotRemoved(ctx context.Context, current, next RolesDocument) error {
	if current.Roles == nil {
		return nil
	}
	removed := make([]string, 0)
	for code := range current.Roles {
		norm := normalizeRoleCode(code)
		if norm == ReservedRoleCode {
			if _, ok := next.Roles[code]; !ok {
				if _, ok2 := next.Roles[norm]; !ok2 {
					return fmt.Errorf("cannot delete builtin role %s", ReservedRoleCode)
				}
			}
			continue
		}
		found := false
		for k := range next.Roles {
			if normalizeRoleCode(k) == norm {
				found = true
				break
			}
		}
		if !found {
			removed = append(removed, norm)
		}
	}
	if len(removed) == 0 {
		return nil
	}
	inUse, err := st.rolesReferencedByUsers(ctx, removed)
	if err != nil {
		return err
	}
	if len(inUse) > 0 {
		return RolesInUseConflict(inUse)
	}
	return nil
}

type roleUserCounter interface {
	CountUsersByRoles(ctx context.Context, roles []string) (map[string]int64, error)
}

var userCounter roleUserCounter

func SetUserRoleCounter(counter roleUserCounter) {
	userCounter = counter
}

func (st *RolesStore) rolesReferencedByUsers(ctx context.Context, roles []string) ([]string, error) {
	if userCounter == nil {
		return nil, nil
	}
	counts, err := userCounter.CountUsersByRoles(ctx, roles)
	if err != nil {
		return nil, err
	}
	inUse := make([]string, 0)
	for role, n := range counts {
		if n > 0 {
			inUse = append(inUse, role)
		}
	}
	return inUse, nil
}

func (st *RolesStore) writeAtomic(raw []byte, doc RolesDocument) error {
	dir := filepath.Dir(st.path)
	if err := validateRolesPath(st.path); err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	marshaled, err := MarshalYAML(doc)
	if err != nil {
		return err
	}
	if len(raw) > 0 && strings.TrimSpace(string(raw)) != "" {
		// Prefer caller body when valid YAML was supplied (preserves comments less, but matches PUT body).
		if _, parseErr := ParseYAML(raw); parseErr == nil {
			// Re-marshal validated doc to keep revision metadata consistent.
			_ = raw
		}
	}
	newPath := st.path + ".new"
	if err := os.WriteFile(newPath, marshaled, 0o644); err != nil {
		return err
	}
	if existing, err := os.ReadFile(st.path); err == nil && len(existing) > 0 {
		_ = os.WriteFile(st.path+".bak", existing, 0o644)
	}
	if err := os.Rename(newPath, st.path); err != nil {
		return err
	}
	return nil
}

func validateRolesPath(path string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	abs = filepath.Clean(abs)
	if os.Getenv("ACCESS_ROLES_PATH_ALLOW_ANY") == "1" {
		return nil
	}
	if strings.HasPrefix(abs, "/etc/ad-event-processor/") {
		return nil
	}
	if strings.Contains(abs, "/deploy/operator/") {
		return nil
	}
	if strings.Contains(abs, "/var/access_test/") {
		return nil
	}
	return fmt.Errorf("roles path not under allowlist: %s", abs)
}

func MarshalYAML(doc RolesDocument) ([]byte, error) {
	raw := rolesYAMLFromDocument(doc)
	return yaml.Marshal(raw)
}

type rolesYAML struct {
	Version   int                            `yaml:"version,omitempty"`
	Revision  int                            `yaml:"revision,omitempty"`
	UpdatedAt string                         `yaml:"updated_at,omitempty"`
	UpdatedBy string                         `yaml:"updated_by,omitempty"`
	Roles     map[string]roleYAML            `yaml:"roles"`
}

type roleYAML struct {
	Scope        string   `yaml:"scope"`
	Builtin      bool     `yaml:"builtin,omitempty"`
	Label        string   `yaml:"label,omitempty"`
	Capabilities []string `yaml:"capabilities,omitempty"`
	Permissions  []string `yaml:"permissions,omitempty"`
}

func (r rolesYAML) toDocument() RolesDocument {
	doc := RolesDocument{
		Version:   r.Version,
		Revision:  r.Revision,
		UpdatedAt: r.UpdatedAt,
		UpdatedBy: r.UpdatedBy,
		Roles:     make(map[string]RoleDefinition, len(r.Roles)),
	}
	for code, role := range r.Roles {
		doc.Roles[normalizeRoleCode(code)] = RoleDefinition{
			Scope:        role.Scope,
			Builtin:      role.Builtin,
			Label:        role.Label,
			Capabilities: role.Capabilities,
			Permissions:  role.Permissions,
		}
	}
	return doc
}

func rolesYAMLFromDocument(doc RolesDocument) rolesYAML {
	out := rolesYAML{
		Version:   doc.Version,
		Revision:  doc.Revision,
		UpdatedAt: doc.UpdatedAt,
		UpdatedBy: doc.UpdatedBy,
		Roles:     make(map[string]roleYAML, len(doc.Roles)),
	}
	for code, role := range doc.Roles {
		out.Roles[code] = roleYAML{
			Scope:        role.Scope,
			Builtin:      role.Builtin,
			Label:        role.Label,
			Capabilities: role.Capabilities,
			Permissions:  role.Permissions,
		}
	}
	return out
}

func viewsFromDoc(doc RolesDocument, compiled map[string][]string, counts map[string]int64) map[string]RoleView {
	out := make(map[string]RoleView, len(doc.Roles))
	for code, role := range doc.Roles {
		norm := normalizeRoleCode(code)
		perms := compiled[norm]
		if perms == nil {
			perms = CompileRolePermissions(role)
		}
		view := RoleView{
			Code:                norm,
			Scope:               strings.ToLower(strings.TrimSpace(role.Scope)),
			Builtin:             role.Builtin || IsBuiltinRole(norm),
			Label:               role.Label,
			Capabilities:        role.Capabilities,
			Permissions:         role.Permissions,
			CompiledPermissions: perms,
		}
		if counts != nil {
			view.MemberCount = counts[norm]
		}
		out[norm] = view
	}
	return out
}

var errRevisionConflict = errors.New("revision_conflict")

func validationError(errs []ValidationDetail) error {
	return &validateErr{details: errs}
}

type validateErr struct {
	details []ValidationDetail
}

func (e *validateErr) Error() string {
	return "validation failed"
}

func IsValidationError(err error) ([]ValidationDetail, bool) {
	if err == nil {
		return nil, false
	}
	if ve, ok := err.(*validateErr); ok {
		return ve.details, true
	}
	return nil, false
}

func IsRevisionConflict(err error) bool {
	return errors.Is(err, errRevisionConflict)
}

func IsPathError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "roles path not under allowlist")
}

func LoadRolesIntoPolicy(path string, policy *authz.Store) error {
	store := NewRolesStore(path, policy)
	if err := store.LoadFromDisk(); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	}
	return nil
}
