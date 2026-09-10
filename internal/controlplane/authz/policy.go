package authz

import (
	"context"
	"sort"
	"sync"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Scope string

const (
	ScopeGlobal   Scope = "global"
	ScopeCustomer Scope = "customer"
	ScopeTeam     Scope = "team"
)

type MaskLevel string

const (
	MaskFull   MaskLevel = "full"
	MaskMasked MaskLevel = "masked"
)

const (
	PermCampaignsRead       = "campaigns:read"
	PermCampaignsReadMasked = "campaigns:read:masked"
	PermCampaignsWrite      = "campaigns:write"
	PermCampaignsWriteMask  = "campaigns:write:masked"
	PermCampaignsPause      = "campaigns:pause"
	PermBillingRead         = "billing:read"
	PermBillingWrite        = "billing:write"
	PermSystemBlacklist     = "blacklist:write"
	PermAccessRead          = "access:read"
	PermAccessWrite         = "access:write"
	PermPostbacksRead       = "postbacks:read"
	PermPostbacksWrite      = "postbacks:write"
	PermCampaignsArchive    = "campaigns:archive"
	PermCampaignsDelete     = "campaigns:delete"
	PermExportsRead         = "exports:read"
	PermExportsRun          = "exports:run"
	PermTeamRead            = "team:read"
	PermTeamWrite           = "team:write"
)

type Snapshot struct {
	Permissions map[string]struct{}
	Mask        MaskLevel
	Scope       Scope
}

func (s Snapshot) Has(permission string) bool {
	if s.Permissions == nil {
		return false
	}
	if _, ok := s.Permissions["*"]; ok {
		return true
	}
	_, ok := s.Permissions[permission]
	return ok
}

func (s Snapshot) HasAny(permissions ...string) bool {
	for _, p := range permissions {
		if s.Has(p) {
			return true
		}
	}
	return false
}

// PermissionsList returns sorted permission keys from a snapshot (including "*" when present).
func PermissionsList(snap Snapshot) []string {
	if len(snap.Permissions) == 0 {
		return nil
	}
	out := make([]string, 0, len(snap.Permissions))
	for p := range snap.Permissions {
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}

// SessionPermissionsList prefers the request snapshot; otherwise resolves from the policy store.
func SessionPermissionsList(ctx context.Context, store *Store, pool *pgxpool.Pool, userID uuid.UUID, role string) []string {
	if snap, ok := SnapshotFromContext(ctx); ok {
		return PermissionsList(snap)
	}
	if store != nil {
		return PermissionsList(store.EffectivePermissionsDB(ctx, pool, userID, role))
	}
	return nil
}

type ctxKey struct{}

var snapshotKey ctxKey

func WithSnapshot(ctx context.Context, snap Snapshot) context.Context {
	return context.WithValue(ctx, snapshotKey, snap)
}

func SnapshotFromContext(ctx context.Context) (Snapshot, bool) {
	v, ok := ctx.Value(snapshotKey).(Snapshot)
	return v, ok
}

func MaskLevelFromPermissions(perms map[string]struct{}) MaskLevel {
	if perms == nil {
		return MaskMasked
	}
	if _, ok := perms["*"]; ok {
		return MaskFull
	}
	if _, ok := perms[PermCampaignsRead]; ok {
		return MaskFull
	}
	return MaskMasked
}

func IsMaskedMutation(ctx context.Context) bool {
	snap, ok := SnapshotFromContext(ctx)
	return ok && snap.Mask == MaskMasked
}

type Store struct {
	mu        sync.RWMutex
	rolePerms map[string]map[string]struct{}
	roleScope map[string]Scope
	userSnaps sync.Map
}

func NewStore() *Store {
	return &Store{
		rolePerms: make(map[string]map[string]struct{}),
		roleScope: make(map[string]Scope),
	}
}

func (st *Store) SetRole(role string, scope Scope, permissions []string) {
	st.mu.Lock()
	defer st.mu.Unlock()
	set := make(map[string]struct{}, len(permissions))
	for _, p := range permissions {
		set[p] = struct{}{}
	}
	st.rolePerms[role] = set
	st.roleScope[role] = scope
}

func (st *Store) EffectivePermissions(userID uuid.UUID, role string) Snapshot {
	if userID != uuid.Nil {
		if v, ok := st.userSnaps.Load(userID); ok {
			return v.(Snapshot)
		}
	}
	snap := st.buildSnapshot(role)
	if userID != uuid.Nil {
		st.userSnaps.Store(userID, snap)
	}
	return snap
}

func (st *Store) buildSnapshot(role string) Snapshot {
	st.mu.RLock()
	perms := st.rolePerms[role]
	scope := st.roleScope[role]
	st.mu.RUnlock()

	copied := make(map[string]struct{}, len(perms))
	for p := range perms {
		copied[p] = struct{}{}
	}
	if scope == "" {
		scope = ScopeCustomer
	}
	return Snapshot{
		Permissions: copied,
		Mask:        MaskLevelFromPermissions(copied),
		Scope:       scope,
	}
}

func (st *Store) RefreshUser(userID uuid.UUID, role string) {
	if userID == uuid.Nil {
		return
	}
	st.userSnaps.Store(userID, st.buildSnapshot(role))
}

func (st *Store) Reload() {
	st.userSnaps = sync.Map{}
}

func (st *Store) ScopeForRole(role string) Scope {
	st.mu.RLock()
	defer st.mu.RUnlock()
	if s, ok := st.roleScope[role]; ok {
		return s
	}
	return ScopeCustomer
}

func (st *Store) RoleExists(role string) bool {
	if st == nil {
		return false
	}
	st.mu.RLock()
	_, ok := st.rolePerms[role]
	st.mu.RUnlock()
	return ok
}

func (st *Store) ListRoles() []string {
	if st == nil {
		return nil
	}
	st.mu.RLock()
	out := make([]string, 0, len(st.rolePerms))
	for role := range st.rolePerms {
		out = append(out, role)
	}
	st.mu.RUnlock()
	sort.Strings(out)
	return out
}
