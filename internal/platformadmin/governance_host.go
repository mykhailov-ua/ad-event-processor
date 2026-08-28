package platformadmin

import (
	"context"

	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type GovernanceHost interface {
	Pool() *pgxpool.Pool
	NormalizeTeamRole(role string) (string, error)
	ErrValidation(msg string) error
	ErrTeamMemberNotFound() error
	ErrCampaignNotFound() error
	GetCampaignRow(ctx context.Context, id uuid.UUID) (db.Campaign, error)
	AuditLog(ctx context.Context, q db.Querier, adminID uuid.UUID, action, targetType string, targetID *uuid.UUID, changes, metadata any)
	ApplyCampaignBudgetPatch(ctx context.Context, q db.Querier, locked db.Campaign, newLimit int64) error
	MapCampaignNotFound(err error) error
}

type Governance struct {
	host GovernanceHost
}

func NewGovernance(host GovernanceHost) *Governance {
	return &Governance{host: host}
}

func (g *Governance) InviteTeamMember(ctx context.Context, customerID uuid.UUID, email, role string) (TeamMemberDTO, error) {
	return inviteTeamMember(ctx, g.host, customerID, email, role)
}

func (g *Governance) UpdateTeamMember(ctx context.Context, customerID, userID uuid.UUID, in UpdateTeamMemberRequest) (TeamMemberDTO, error) {
	return updateTeamMember(ctx, g.host, customerID, userID, in)
}

func (g *Governance) ListTeamBudgetApprovals(ctx context.Context, customerID uuid.UUID) ([]TeamBudgetApprovalDTO, error) {
	return listTeamBudgetApprovals(ctx, g.host.Pool(), customerID)
}

func (g *Governance) ResolveTeamBudgetApproval(ctx context.Context, customerID, approvalID, resolverID uuid.UUID, approve bool) error {
	return resolveTeamBudgetApproval(ctx, g.host, customerID, approvalID, resolverID, approve)
}

func AssignCampaignOwner(ctx context.Context, host GovernanceHost, campaignID, ownerUserID uuid.UUID) error {
	camp, err := host.GetCampaignRow(ctx, campaignID)
	if err != nil {
		return err
	}
	pool := host.Pool()
	if pool == nil {
		return errTeamServiceUnavailable()
	}
	var memberCustomer uuid.UUID
	err = pool.QueryRow(ctx, `SELECT customer_id FROM users WHERE id = $1`, ownerUserID).Scan(&memberCustomer)
	if err != nil {
		if err == pgx.ErrNoRows {
			return host.ErrTeamMemberNotFound()
		}
		return err
	}
	if memberCustomer != uuid.UUID(camp.CustomerID.Bytes) {
		return host.ErrValidation("owner must belong to campaign customer")
	}
	_, err = pool.Exec(ctx, `UPDATE campaigns SET owner_user_id = $1 WHERE id = $2`, ownerUserID, campaignID)
	return err
}
