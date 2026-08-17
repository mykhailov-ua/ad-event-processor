package controlplane

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"strings"
	"time"

	"github.com/bidshard/ad-event-processor/internal/controlplane/adminapi"
	db "github.com/bidshard/ad-event-processor/internal/domain/db"
	"github.com/bidshard/ad-event-processor/internal/identity"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func normalizeTeamMemberRole(role string) (string, error) {
	switch NormalizeRole(role) {
	case RoleTeamLead, RoleMediaBuyer:
		return NormalizeRole(role), nil
	default:
		return "", errValidation("role must be TL or MB")
	}
}

func (s *Service) InviteTeamMember(ctx context.Context, customerID uuid.UUID, email, role string) (adminapi.TeamMemberDTO, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" {
		return adminapi.TeamMemberDTO{}, errValidation("email required")
	}
	role, err := normalizeTeamMemberRole(role)
	if err != nil {
		return adminapi.TeamMemberDTO{}, err
	}
	pool := s.GetPool()
	if pool == nil {
		return adminapi.TeamMemberDTO{}, errors.New("team service unavailable")
	}

	hasher, err := identity.NewPasswordHasher(32768, 2, 2)
	if err != nil {
		return adminapi.TeamMemberDTO{}, err
	}
	tempPwd := make([]byte, 24)
	if _, err := rand.Read(tempPwd); err != nil {
		return adminapi.TeamMemberDTO{}, err
	}
	password := base64.RawURLEncoding.EncodeToString(tempPwd)
	hash, err := hasher.HashPassword(password)
	if err != nil {
		return adminapi.TeamMemberDTO{}, err
	}

	userID := uuid.New()
	_, err = pool.Exec(ctx, `
		INSERT INTO users (id, email, password_hash, role, customer_id, email_verified)
		VALUES ($1, $2, $3, $4, $5, TRUE)
		ON CONFLICT (email) DO NOTHING`,
		userID, email, hash, role, customerID)
	if err != nil {
		return adminapi.TeamMemberDTO{}, err
	}
	var gotID uuid.UUID
	err = pool.QueryRow(ctx, `SELECT id FROM users WHERE email = $1 AND customer_id = $2`, email, customerID).Scan(&gotID)
	if err != nil {
		return adminapi.TeamMemberDTO{}, err
	}
	return s.teamMemberDTO(ctx, customerID, gotID)
}

type UpdateTeamMemberInput struct {
	Role          *string
	IsBlocked     *bool
	SpendCapMicro *int64
}

func (s *Service) UpdateTeamMember(ctx context.Context, customerID, userID uuid.UUID, in adminapi.UpdateTeamMemberRequest) (adminapi.TeamMemberDTO, error) {
	pool := s.GetPool()
	if pool == nil {
		return adminapi.TeamMemberDTO{}, errors.New("team service unavailable")
	}
	var exists bool
	err := pool.QueryRow(ctx, `SELECT TRUE FROM users WHERE id = $1 AND customer_id = $2`, userID, customerID).Scan(&exists)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return adminapi.TeamMemberDTO{}, ErrTeamMemberNotFound
		}
		return adminapi.TeamMemberDTO{}, err
	}
	if in.Role != nil {
		role, err := normalizeTeamMemberRole(*in.Role)
		if err != nil {
			return adminapi.TeamMemberDTO{}, err
		}
		if _, err := pool.Exec(ctx, `UPDATE users SET role = $1, updated_at = NOW() WHERE id = $2`, role, userID); err != nil {
			return adminapi.TeamMemberDTO{}, err
		}
	}
	if in.IsBlocked != nil {
		if _, err := pool.Exec(ctx, `UPDATE users SET is_blocked = $1, updated_at = NOW() WHERE id = $2`, *in.IsBlocked, userID); err != nil {
			return adminapi.TeamMemberDTO{}, err
		}
	}
	if in.SpendCapMicro != nil {
		if *in.SpendCapMicro < 0 {
			return adminapi.TeamMemberDTO{}, errValidation("spend_cap_micro must be >= 0")
		}
		_, err := pool.Exec(ctx, `
			INSERT INTO team_member_limits (user_id, customer_id, spend_cap_micro, updated_at)
			VALUES ($1, $2, $3, NOW())
			ON CONFLICT (user_id) DO UPDATE SET spend_cap_micro = EXCLUDED.spend_cap_micro, updated_at = NOW()`,
			userID, customerID, *in.SpendCapMicro)
		if err != nil {
			return adminapi.TeamMemberDTO{}, err
		}
	}
	return s.teamMemberDTO(ctx, customerID, userID)
}

func (s *Service) teamMemberDTO(ctx context.Context, customerID, userID uuid.UUID) (adminapi.TeamMemberDTO, error) {
	pool := s.GetPool()
	var m adminapi.TeamMemberDTO
	var created time.Time
	var blocked bool
	err := pool.QueryRow(ctx, `
		SELECT u.id, u.email, u.role, u.created_at, u.is_blocked,
			(SELECT COUNT(*)::bigint FROM campaigns c WHERE c.customer_id = u.customer_id AND c.owner_user_id = u.id),
			COALESCE(l.spend_cap_micro, 0)
		FROM users u
		LEFT JOIN team_member_limits l ON l.user_id = u.id
		WHERE u.id = $1 AND u.customer_id = $2`,
		userID, customerID).Scan(&userID, &m.Email, &m.Role, &created, &blocked, &m.CampaignsOwned, &m.SpendCapMicro)
	if err != nil {
		return adminapi.TeamMemberDTO{}, err
	}
	m.UserID = userID.String()
	m.CreatedAt = created.UTC().Format(time.RFC3339)
	m.IsBlocked = blocked
	return m, nil
}

func (s *Service) AssignCampaignOwner(ctx context.Context, campaignID, ownerUserID uuid.UUID) error {
	camp, err := s.GetCampaignRow(ctx, campaignID)
	if err != nil {
		return err
	}
	pool := s.GetPool()
	var memberCustomer uuid.UUID
	err = pool.QueryRow(ctx, `SELECT customer_id FROM users WHERE id = $1`, ownerUserID).Scan(&memberCustomer)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrTeamMemberNotFound
		}
		return err
	}
	if memberCustomer != uuid.UUID(camp.CustomerID.Bytes) {
		return errValidation("owner must belong to campaign customer")
	}
	_, err = pool.Exec(ctx, `UPDATE campaigns SET owner_user_id = $1 WHERE id = $2`, ownerUserID, campaignID)
	return err
}

func (s *Service) createBudgetApprovalPending(
	ctx context.Context,
	customerID, userID, campaignID uuid.UUID,
	previousLimit, requestedLimit int64,
) (uuid.UUID, error) {
	pool := s.GetPool()
	id := uuid.New()
	_, err := pool.Exec(ctx, `
		INSERT INTO team_budget_approvals (
			id, customer_id, user_id, campaign_id,
			requested_budget_micro, previous_budget_micro, status
		) VALUES ($1, $2, $3, $4, $5, $6, 'PENDING')`,
		id, customerID, userID, campaignID, requestedLimit, previousLimit)
	return id, err
}

func (s *Service) ListTeamBudgetApprovals(ctx context.Context, customerID uuid.UUID) ([]adminapi.TeamBudgetApprovalDTO, error) {
	pool := s.GetPool()
	rows, err := pool.Query(ctx, `
		SELECT id, user_id, campaign_id, requested_budget_micro, previous_budget_micro, status, created_at
		FROM team_budget_approvals
		WHERE customer_id = $1 AND status = 'PENDING'
		ORDER BY created_at DESC`, customerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]adminapi.TeamBudgetApprovalDTO, 0, 8)
	for rows.Next() {
		var row adminapi.TeamBudgetApprovalDTO
		var id, userID, campaignID uuid.UUID
		var created time.Time
		if err := rows.Scan(&id, &userID, &campaignID, &row.RequestedBudgetMicro, &row.PreviousBudgetMicro, &row.Status, &created); err != nil {
			return nil, err
		}
		row.ID = id.String()
		row.UserID = userID.String()
		row.CampaignID = campaignID.String()
		row.CreatedAt = created.UTC().Format(time.RFC3339)
		out = append(out, row)
	}
	return out, rows.Err()
}

func (s *Service) ResolveTeamBudgetApproval(ctx context.Context, customerID, approvalID, resolverID uuid.UUID, approve bool) error {
	pool := s.GetPool()
	var userID, campaignID uuid.UUID
	var requested, previous int64
	var status string
	err := pool.QueryRow(ctx, `
		SELECT user_id, campaign_id, requested_budget_micro, previous_budget_micro, status
		FROM team_budget_approvals
		WHERE id = $1 AND customer_id = $2`, approvalID, customerID).
		Scan(&userID, &campaignID, &requested, &previous, &status)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrTeamMemberNotFound
		}
		return err
	}
	if status != "PENDING" {
		return errValidation("approval already resolved")
	}
	nextStatus := "DENIED"
	if approve {
		nextStatus = "APPROVED"
	}
	_, err = pool.Exec(ctx, `
		UPDATE team_budget_approvals
		SET status = $1, resolved_at = NOW(), resolved_by = $2
		WHERE id = $3`, nextStatus, resolverID, approvalID)
	if err != nil {
		return err
	}
	if !approve {
		return nil
	}
	return pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		q := db.New(tx)
		locked, err := q.GetCampaignForUpdate(ctx, pgtype.UUID{Bytes: campaignID, Valid: true})
		if err != nil {
			return mapNotFound(err, ErrCampaignNotFound)
		}
		return s.applyCampaignBudgetPatch(ctx, q, locked, requested)
	})
}
