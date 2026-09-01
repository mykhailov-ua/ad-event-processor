package platformadmin

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"strings"
	"time"

	"ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/identity"
	"ad-event-processor/pkg/coldpath"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type auditBudgetApprovalDeny struct {
	CampaignID           string `json:"campaign_id"`
	RequestedBudgetMicro int64  `json:"requested_budget_micro"`
	PreviousBudgetMicro  int64  `json:"previous_budget_micro"`
}

func errTeamServiceUnavailable() error {
	return errors.New("team service unavailable")
}

func inviteTeamMember(ctx context.Context, host GovernanceHost, customerID uuid.UUID, email, role string) (TeamMemberDTO, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" {
		return TeamMemberDTO{}, host.ErrValidation("email required")
	}
	normalizedRole, err := host.NormalizeTeamRole(role)
	if err != nil {
		return TeamMemberDTO{}, err
	}
	pool := host.Pool()
	if pool == nil {
		return TeamMemberDTO{}, errTeamServiceUnavailable()
	}

	hasher, err := identity.NewPasswordHasher(32768, 2, 2)
	if err != nil {
		return TeamMemberDTO{}, err
	}
	tempPwd := make([]byte, 24)
	if _, err := rand.Read(tempPwd); err != nil {
		return TeamMemberDTO{}, err
	}
	password := base64.RawURLEncoding.EncodeToString(tempPwd)
	hash, err := hasher.HashPassword(password)
	if err != nil {
		return TeamMemberDTO{}, err
	}

	userID := uuid.New()
	_, err = pool.Exec(ctx, `
		INSERT INTO users (id, email, password_hash, role, customer_id, email_verified)
		VALUES ($1, $2, $3, $4, $5, FALSE)
		ON CONFLICT (email) DO NOTHING`,
		userID, email, hash, normalizedRole, customerID)
	if err != nil {
		return TeamMemberDTO{}, err
	}
	var gotID uuid.UUID
	err = pool.QueryRow(ctx, `SELECT id FROM users WHERE email = $1 AND customer_id = $2`, email, customerID).Scan(&gotID)
	if err != nil {
		return TeamMemberDTO{}, err
	}

	rdb := host.InviteRedis()
	if rdb == nil {
		return TeamMemberDTO{}, errTeamServiceUnavailable()
	}
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return TeamMemberDTO{}, err
	}
	token := base64.RawURLEncoding.EncodeToString(tokenBytes)
	if err := StoreTeamInvite(ctx, rdb, token, TeamInvitePayload{
		UserID:     gotID,
		CustomerID: customerID,
		Email:      email,
	}); err != nil {
		return TeamMemberDTO{}, err
	}
	baseURL := host.PublicPanelBaseURL(PanelRequestFromContext(ctx))
	acceptURL := strings.TrimRight(baseURL, "/") + "/invite/accept?token=" + token
	host.EnqueueInviteEmail(ctx, email, acceptURL)

	return teamMemberDTO(ctx, host, customerID, gotID)
}

func updateTeamMember(ctx context.Context, host GovernanceHost, customerID, userID uuid.UUID, in UpdateTeamMemberRequest) (TeamMemberDTO, error) {
	pool := host.Pool()
	if pool == nil {
		return TeamMemberDTO{}, errTeamServiceUnavailable()
	}
	var exists bool
	err := pool.QueryRow(ctx, `SELECT TRUE FROM users WHERE id = $1 AND customer_id = $2`, userID, customerID).Scan(&exists)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return TeamMemberDTO{}, host.ErrTeamMemberNotFound()
		}
		return TeamMemberDTO{}, err
	}
	if in.Role != nil {
		role, err := host.NormalizeTeamRole(*in.Role)
		if err != nil {
			return TeamMemberDTO{}, err
		}
		if _, err := pool.Exec(ctx, `UPDATE users SET role = $1, updated_at = NOW() WHERE id = $2`, role, userID); err != nil {
			return TeamMemberDTO{}, err
		}
	}
	if in.IsBlocked != nil {
		if _, err := pool.Exec(ctx, `UPDATE users SET is_blocked = $1, updated_at = NOW() WHERE id = $2`, *in.IsBlocked, userID); err != nil {
			return TeamMemberDTO{}, err
		}
	}
	if in.SpendCapMicro != nil {
		if *in.SpendCapMicro < 0 {
			return TeamMemberDTO{}, host.ErrValidation("spend_cap_micro must be >= 0")
		}
		_, err := pool.Exec(ctx, `
			INSERT INTO team_member_limits (user_id, customer_id, spend_cap_micro, updated_at)
			VALUES ($1, $2, $3, NOW())
			ON CONFLICT (user_id) DO UPDATE SET spend_cap_micro = EXCLUDED.spend_cap_micro, updated_at = NOW()`,
			userID, customerID, *in.SpendCapMicro)
		if err != nil {
			return TeamMemberDTO{}, err
		}
	}
	return teamMemberDTO(ctx, host, customerID, userID)
}

func teamMemberDTO(ctx context.Context, host GovernanceHost, customerID, userID uuid.UUID) (TeamMemberDTO, error) {
	pool := host.Pool()
	var m TeamMemberDTO
	var created time.Time
	var blocked bool
	err := pool.QueryRow(ctx, `
		SELECT u.id, u.email, u.role, u.created_at, u.is_blocked,
			COALESCE(cc.campaigns_owned, 0),
			COALESCE(l.spend_cap_micro, 0)
		FROM users u
		LEFT JOIN team_member_limits l ON l.user_id = u.id
		LEFT JOIN (
			SELECT owner_user_id, COUNT(*)::bigint AS campaigns_owned
			FROM campaigns
			WHERE customer_id = $2
			GROUP BY owner_user_id
		) cc ON cc.owner_user_id = u.id
		WHERE u.id = $1 AND u.customer_id = $2`,
		userID, customerID).Scan(&userID, &m.Email, &m.Role, &created, &blocked, &m.CampaignsOwned, &m.SpendCapMicro)
	if err != nil {
		return TeamMemberDTO{}, err
	}
	m.UserID = userID.String()
	m.CreatedAt = created.UTC().Format(time.RFC3339)
	m.CreatedAtDisplay = coldpath.RFC3339Display(m.CreatedAt)
	m.IsBlocked = blocked
	return m, nil
}

const (
	TeamBudgetApprovalsDefaultLimit = 50
	TeamBudgetApprovalsMaxLimit     = 100
)

func normalizeTeamBudgetApprovalsLimit(limit int) int {
	if limit <= 0 {
		return TeamBudgetApprovalsDefaultLimit
	}
	if limit > TeamBudgetApprovalsMaxLimit {
		return TeamBudgetApprovalsMaxLimit
	}
	return limit
}

func normalizeTeamBudgetApprovalsOffset(offset int) int {
	if offset < 0 {
		return 0
	}
	return offset
}

func listTeamBudgetApprovals(ctx context.Context, pool *pgxpool.Pool, customerID uuid.UUID, limit, offset int) ([]TeamBudgetApprovalDTO, int64, error) {
	if pool == nil {
		return nil, 0, errTeamServiceUnavailable()
	}
	limit = normalizeTeamBudgetApprovalsLimit(limit)
	offset = normalizeTeamBudgetApprovalsOffset(offset)

	var total int64
	if err := pool.QueryRow(ctx, `
		SELECT count(*)
		FROM team_budget_approvals
		WHERE customer_id = $1 AND status = 'PENDING'`, customerID).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := pool.Query(ctx, `
		SELECT id, user_id, campaign_id, requested_budget_micro, previous_budget_micro, status, created_at
		FROM team_budget_approvals
		WHERE customer_id = $1 AND status = 'PENDING'
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`, customerID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := make([]TeamBudgetApprovalDTO, 0, limit)
	for rows.Next() {
		var row TeamBudgetApprovalDTO
		var id, userID, campaignID uuid.UUID
		var created time.Time
		if err := rows.Scan(&id, &userID, &campaignID, &row.RequestedBudgetMicro, &row.PreviousBudgetMicro, &row.Status, &created); err != nil {
			return nil, 0, err
		}
		row.ID = id.String()
		row.UserID = userID.String()
		row.CampaignID = campaignID.String()
		row.CreatedAt = created.UTC().Format(time.RFC3339)
		row.CreatedAtDisplay = coldpath.RFC3339Display(row.CreatedAt)
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return out, total, nil
}

func resolveTeamBudgetApproval(ctx context.Context, host GovernanceHost, customerID, approvalID, resolverID uuid.UUID, approve bool) error {
	pool := host.Pool()
	if pool == nil {
		return errTeamServiceUnavailable()
	}
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
			return host.ErrTeamMemberNotFound()
		}
		return err
	}
	if status != "PENDING" {
		return host.ErrValidation("approval already resolved")
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
		host.AuditLog(ctx, nil, resolverID, "DENY_BUDGET_APPROVAL", "team_budget_approval", &approvalID, auditBudgetApprovalDeny{
			CampaignID:           campaignID.String(),
			RequestedBudgetMicro: requested,
			PreviousBudgetMicro:  previous,
		}, nil)
		return nil
	}
	return pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		q := db.New(tx)
		locked, err := q.GetCampaignForUpdate(ctx, pgtype.UUID{Bytes: campaignID, Valid: true})
		if err != nil {
			return host.MapCampaignNotFound(err)
		}
		return host.ApplyCampaignBudgetPatch(ctx, q, locked, requested)
	})
}
