package authz

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func LoadUserPermissionsFromDB(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (map[string]struct{}, error) {
	if pool == nil || userID == uuid.Nil {
		return nil, nil
	}
	rows, err := pool.Query(ctx, `
SELECT DISTINCT p.permission
FROM auth.user_roles ur
JOIN auth.role_permissions rp ON rp.role_id = ur.role_id
JOIN auth.permissions p ON p.id = rp.permission_id
WHERE ur.user_id = $1`, userID)
	if err != nil {
		if strings.Contains(err.Error(), "does not exist") {
			return nil, nil
		}
		return nil, err
	}
	defer rows.Close()

	perms := make(map[string]struct{})
	for rows.Next() {
		var perm string
		if err := rows.Scan(&perm); err != nil {
			return nil, err
		}
		perms[perm] = struct{}{}
	}
	return perms, rows.Err()
}

func (st *Store) EffectivePermissionsDB(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, role string) Snapshot {
	snap := st.EffectivePermissions(userID, role)
	if pool == nil || userID == uuid.Nil {
		return snap
	}
	dbPerms, err := LoadUserPermissionsFromDB(ctx, pool, userID)
	if err != nil || len(dbPerms) == 0 {
		return snap
	}
	merged := make(map[string]struct{}, len(snap.Permissions)+len(dbPerms))
	for p := range snap.Permissions {
		merged[p] = struct{}{}
	}
	for p := range dbPerms {
		merged[p] = struct{}{}
	}
	snap = Snapshot{
		Permissions: merged,
		Mask:        MaskLevelFromPermissions(merged),
		Scope:       st.ScopeForRole(role),
	}
	st.userSnaps.Store(userID, snap)
	return snap
}
