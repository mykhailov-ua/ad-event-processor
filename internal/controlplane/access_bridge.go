package controlplane

import (
	"context"
	"fmt"

	"ad-event-processor/internal/access"
	ctrlhttp "ad-event-processor/internal/control/http"
	"ad-event-processor/internal/controlplane/authz"
	"ad-event-processor/internal/platformadmin"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type accessRolesReloader struct {
	store *access.RolesStore
}

func (r accessRolesReloader) ReloadRoles() error {
	if r.store == nil {
		return fmt.Errorf("roles store not configured")
	}
	return r.store.Reload()
}

func (r accessRolesReloader) RolesPath() string {
	if r.store == nil {
		return authz.DefaultRolesPath()
	}
	return r.store.Path()
}

type userRoleCounter struct {
	pool *pgxpool.Pool
}

func (c userRoleCounter) CountUsersByRoles(ctx context.Context, roles []string) (map[string]int64, error) {
	if c.pool == nil || len(roles) == 0 {
		return map[string]int64{}, nil
	}
	out := make(map[string]int64, len(roles))
	for _, role := range roles {
		var count int64
		err := c.pool.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE role = $1`, role).Scan(&count)
		if err != nil {
			return nil, err
		}
		out[role] = count
	}
	return out, nil
}

func wireAccessRolesStore(path string, policy *authz.Store, pool *pgxpool.Pool) *access.RolesStore {
	access.SetUserRoleCounter(userRoleCounter{pool: pool})
	store := access.NewRolesStore(path, policy)
	_ = store.LoadFromDisk()
	return store
}

func (s *Service) SetRolesDocumentSource(doc func() access.RolesDocument) {
	if s == nil {
		return
	}
	s.rolesDocumentSource = doc
}

func (s *Service) ValidateAssignableRole(role string) (string, error) {
	if s != nil && s.rolesDocumentSource != nil {
		return access.ValidateAssignableTeamRole(s.rolesDocumentSource(), role)
	}
	return s.normalizeTeamRoleLegacy(role)
}

func (s *Service) normalizeTeamRoleLegacy(role string) (string, error) {
	normalized := ctrlhttp.NormalizeRole(role)
	switch normalized {
	case ctrlhttp.RoleTeamLead, ctrlhttp.RoleMediaBuyer, ctrlhttp.RoleBuyer:
		return normalized, nil
	default:
		return "", errValidation("role must be TL, MB, or B")
	}
}

func (s *Service) auditAccessApply(ctx context.Context, actorID uuid.UUID, changes, metadata any) {
	if s == nil {
		return
	}
	platformadmin.AuditLog(ctx, s, nil, actorID, "access_roles_apply", "access_roles", nil, changes, metadata)
}
