package teamscope

import (
	"context"
	"errors"
	"os"
	"strings"

	"ad-event-processor/internal/controlplane/authz"
	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ListScope carries owner filters applied to campaign list queries.
type ListScope struct {
	OwnerUserID  pgtype.UUID
	OwnerUserIDs []uuid.UUID
}

func teamEnforceDefault() bool {
	return strings.EqualFold(strings.TrimSpace(os.Getenv("TEAM_ENFORCE_OWNERSHIP_DEFAULT")), "true")
}

func customerEnforceOwnership(ctx context.Context, pool *pgxpool.Pool, customerID uuid.UUID) bool {
	if teamEnforceDefault() {
		return true
	}
	if pool == nil || customerID == uuid.Nil {
		return false
	}
	var enforce bool
	err := pool.QueryRow(ctx, `SELECT team_enforce_ownership FROM customers WHERE id = $1`, customerID).Scan(&enforce)
	if err != nil {
		return false
	}
	return enforce
}

func userTeamID(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (uuid.UUID, bool) {
	if pool == nil || userID == uuid.Nil {
		return uuid.Nil, false
	}
	var teamID pgtype.UUID
	err := pool.QueryRow(ctx, `SELECT team_id FROM users WHERE id = $1`, userID).Scan(&teamID)
	if err != nil || !teamID.Valid {
		return uuid.Nil, false
	}
	return uuid.UUID(teamID.Bytes), true
}

func teamMemberUserIDs(ctx context.Context, pool *pgxpool.Pool, customerID, teamID uuid.UUID) ([]uuid.UUID, error) {
	if pool == nil || teamID == uuid.Nil || customerID == uuid.Nil {
		return nil, nil
	}
	rows, err := pool.Query(ctx, `
		SELECT id FROM users
		WHERE team_id = $1 AND customer_id = $2`,
		teamID, customerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// ResolveListScope returns campaign owner filters for the authenticated actor.
func ResolveListScope(ctx context.Context, pool *pgxpool.Pool, customerID uuid.UUID) (ListScope, error) {
	u, ok := authz.GetUser(ctx)
	if !ok || u.UserID == uuid.Nil {
		return ListScope{}, nil
	}
	role := authz.NormalizeRole(u.Role)
	switch role {
	case authz.RoleMediaBuyer:
		return ListScope{OwnerUserID: domain.ToUUID(u.UserID)}, nil
	case authz.RoleBuyer:
		if customerEnforceOwnership(ctx, pool, customerID) {
			return ListScope{OwnerUserID: domain.ToUUID(u.UserID)}, nil
		}
		return ListScope{}, nil
	case authz.RoleTeamLead:
		teamID, hasTeam := userTeamID(ctx, pool, u.UserID)
		if !hasTeam {
			return ListScope{}, nil
		}
		memberIDs, err := teamMemberUserIDs(ctx, pool, customerID, teamID)
		if err != nil {
			return ListScope{}, err
		}
		return ListScope{OwnerUserIDs: memberIDs}, nil
	default:
		snap, snapOK := authz.SnapshotFromContext(ctx)
		if snapOK && snap.Scope == authz.ScopeTeam && snap.Mask == authz.MaskMasked {
			if customerEnforceOwnership(ctx, pool, customerID) {
				return ListScope{OwnerUserID: domain.ToUUID(u.UserID)}, nil
			}
		}
		return ListScope{}, nil
	}
}

var ErrForbidden = errors.New("forbidden")

func ownerUserIDsParam(ids []uuid.UUID) []pgtype.UUID {
	if len(ids) == 0 {
		return nil
	}
	out := make([]pgtype.UUID, len(ids))
	for i, id := range ids {
		out[i] = domain.ToUUID(id)
	}
	return out
}

// OwnerUserIDsParam converts scope member IDs for sqlc uuid[] params.
func OwnerUserIDsParam(ids []uuid.UUID) []pgtype.UUID {
	return ownerUserIDsParam(ids)
}

// AssertCampaignAccess enforces ownership for MB, enforced B, and TL team scope.
func AssertCampaignAccess(ctx context.Context, pool *pgxpool.Pool, camp db.Campaign) error {
	u, ok := authz.GetUser(ctx)
	if !ok {
		return nil
	}
	if !camp.OwnerUserID.Valid {
		return ErrForbidden
	}
	ownerID := uuid.UUID(camp.OwnerUserID.Bytes)
	role := authz.NormalizeRole(u.Role)
	switch role {
	case authz.RoleMediaBuyer:
		if ownerID != u.UserID {
			return ErrForbidden
		}
		return nil
	case authz.RoleBuyer:
		if !customerEnforceOwnership(ctx, pool, u.CustomerID) {
			return nil
		}
		if ownerID != u.UserID {
			return ErrForbidden
		}
		return nil
	case authz.RoleTeamLead:
		teamID, hasTeam := userTeamID(ctx, pool, u.UserID)
		if !hasTeam {
			return nil
		}
		ownerTeamID, ownerHasTeam := userTeamID(ctx, pool, ownerID)
		if !ownerHasTeam || ownerTeamID != teamID {
			return ErrForbidden
		}
		return nil
	default:
		snap, snapOK := authz.SnapshotFromContext(ctx)
		if snapOK && snap.Scope == authz.ScopeTeam && snap.Mask == authz.MaskMasked {
			if customerEnforceOwnership(ctx, pool, u.CustomerID) && ownerID != u.UserID {
				return ErrForbidden
			}
		}
		return nil
	}
}

// ScopedCampaignIDsQuery builds campaign IDs visible to the actor for a customer.
func ScopedCampaignIDsQuery(ctx context.Context, pool *pgxpool.Pool, customerID uuid.UUID) ([]uuid.UUID, error) {
	if pool == nil || customerID == uuid.Nil {
		return nil, nil
	}
	scope, err := ResolveListScope(ctx, pool, customerID)
	if err != nil {
		return nil, err
	}
	if scope.OwnerUserID.Valid {
		rows, err := pool.Query(ctx, `
			SELECT id FROM campaigns
			WHERE customer_id = $1 AND deleted_at IS NULL AND owner_user_id = $2`,
			customerID, scope.OwnerUserID)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		return scanUUIDRows(rows)
	}
	if len(scope.OwnerUserIDs) > 0 {
		rows, err := pool.Query(ctx, `
			SELECT id FROM campaigns
			WHERE customer_id = $1 AND deleted_at IS NULL AND owner_user_id = ANY($2::uuid[])`,
			customerID, scope.OwnerUserIDs)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		return scanUUIDRows(rows)
	}
	rows, err := pool.Query(ctx, `
		SELECT id FROM campaigns
		WHERE customer_id = $1 AND deleted_at IS NULL`,
		customerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanUUIDRows(rows)
}

func scanUUIDRows(rows pgx.Rows) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// AllowOwnerQueryOverride returns false when ScopeTeam actors must not pass owner_user_id.
func AllowOwnerQueryOverride(ctx context.Context) bool {
	u, ok := authz.GetUser(ctx)
	if !ok {
		return true
	}
	snap, snapOK := authz.SnapshotFromContext(ctx)
	if !snapOK || snap.Scope != authz.ScopeTeam {
		return true
	}
	if snap.Has(authz.PermCampaignsWrite) {
		return true
	}
	role := authz.NormalizeRole(u.Role)
	return role != authz.RoleTeamLead && role != authz.RoleMediaBuyer && role != authz.RoleBuyer
}
