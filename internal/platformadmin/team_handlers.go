package platformadmin

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	ctrlhttp "ad-event-processor/internal/control/http"
	"ad-event-processor/internal/controlplane/authz"
	"ad-event-processor/internal/database"
	"ad-event-processor/internal/reports"
	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TeamMemberDTO struct {
	UserID           string `json:"user_id"`
	Email            string `json:"email"`
	Role             string `json:"role"`
	TeamID           string `json:"team_id,omitempty"`
	CampaignsOwned   int64  `json:"campaigns_owned"`
	CreatedAt        string `json:"created_at,omitempty"`
	CreatedAtDisplay string `json:"created_at_display,omitempty"`
	IsBlocked        bool   `json:"is_blocked,omitempty"`
	SpendCapMicro    int64  `json:"spend_cap_micro,omitempty"`
}

type TeamBudgetApprovalDTO struct {
	ID                   string `json:"id"`
	UserID               string `json:"user_id"`
	CampaignID           string `json:"campaign_id"`
	RequestedBudgetMicro int64  `json:"requested_budget_micro"`
	PreviousBudgetMicro  int64  `json:"previous_budget_micro"`
	Status               string `json:"status"`
	CreatedAt            string `json:"created_at,omitempty"`
	CreatedAtDisplay     string `json:"created_at_display,omitempty"`
}

type TeamBudgetApprovalsListResponse struct {
	Items  []TeamBudgetApprovalDTO `json:"items"`
	Total  int64                   `json:"total"`
	Limit  int                     `json:"limit"`
	Offset int                     `json:"offset"`
}

type TeamMembersListResponse struct {
	Items  []TeamMemberDTO `json:"items"`
	Total  int64           `json:"total"`
	Limit  int             `json:"limit"`
	Offset int             `json:"offset"`
}

const (
	TeamMembersDefaultLimit = TeamBudgetApprovalsDefaultLimit
	TeamMembersMaxLimit     = TeamBudgetApprovalsMaxLimit
)

func normalizeTeamMembersLimit(limit int) int {
	return normalizeTeamBudgetApprovalsLimit(limit)
}

func normalizeTeamMembersOffset(offset int) int {
	return normalizeTeamBudgetApprovalsOffset(offset)
}

type InviteTeamMemberRequest struct {
	Email  string  `json:"email"`
	Role   string  `json:"role"`
	TeamID *string `json:"team_id,omitempty"`
}

type UpdateTeamMemberRequest struct {
	Role          *string `json:"role,omitempty"`
	TeamID        *string `json:"team_id,omitempty"`
	IsBlocked     *bool   `json:"is_blocked,omitempty"`
	SpendCapMicro *int64  `json:"spend_cap_micro,omitempty"`
}

type AssignCampaignOwnerRequest struct {
	UserID string `json:"user_id"`
}

type TeamLicenseDTO struct {
	State      string `json:"state"`
	ValidUntil string `json:"valid_until,omitempty"`
	PlanCode   string `json:"plan_code,omitempty"`
}

type TeamOverviewDTO struct {
	CustomerID            string          `json:"customer_id"`
	CustomerName          string          `json:"customer_name"`
	CostCenter            string          `json:"cost_center,omitempty"`
	BalanceMicro          int64           `json:"balance_micro,omitempty"`
	Currency              string          `json:"currency,omitempty"`
	License               *TeamLicenseDTO `json:"license,omitempty"`
	Members               []TeamMemberDTO `json:"members"`
	PendingApprovalsCount int64           `json:"pending_approvals_count,omitempty"`
	PendingForMeCount     int64           `json:"pending_for_me_count,omitempty"`
}

type TeamOverviewReader interface {
	GetTeamOverview(ctx context.Context, customerID uuid.UUID, includeBalance, includeLicense bool, actorUserID uuid.UUID) (TeamOverviewDTO, error)
	ListTeamMembers(ctx context.Context, customerID uuid.UUID, limit, offset int) ([]TeamMemberDTO, int64, error)
	GetTeamMetrics(ctx context.Context, customerID uuid.UUID, from, to time.Time) (TeamMetricsResponse, error)
}

type TeamHTTPHandlers struct {
	Pool                 *pgxpool.Pool
	Team                 TeamOverviewReader
	Governance           TeamGovernance
	ApplyRateLimit       func(http.HandlerFunc) http.HandlerFunc
	RequireAnyPermission func([]string, http.HandlerFunc) http.HandlerFunc
	RequireTeamWrite     func(http.HandlerFunc) http.HandlerFunc
	ResolveCustomerID    func(*http.Request, *uuid.UUID) (uuid.UUID, error)
	SnapshotFromRequest  func(*http.Request) (authz.Snapshot, bool)
	ActorUserID          func(*http.Request) (uuid.UUID, bool)
	PolicyRefresh        ctrlhttp.LoginPolicyRefresher
	WriteServiceError    func(http.ResponseWriter, error)
}

func (h *TeamHTTPHandlers) Register(mux *http.ServeMux) {
	if h == nil || h.Team == nil {
		return
	}
	limit := h.ApplyRateLimit
	perm := h.RequireAnyPermission
	if limit == nil {
		limit = func(next http.HandlerFunc) http.HandlerFunc { return next }
	}
	if perm == nil {
		perm = func(_ []string, next http.HandlerFunc) http.HandlerFunc { return next }
	}
	mux.HandleFunc("GET /api/v1/team/overview", limit(perm(
		[]string{"team:read", "campaigns:read", "billing:read"},
		h.getOverview,
	)))
	mux.HandleFunc("GET /api/v1/team/metrics", limit(perm(
		[]string{"team:read"},
		h.getTeamMetrics,
	)))
	mux.HandleFunc("GET /api/v1/team/budget-approvals/mine", limit(h.listMyBudgetApprovals))
	h.registerTeamGovernanceRoutes(mux, limit, perm)
	h.registerTeamsRoutes(mux, limit, perm)
}

func (h *TeamHTTPHandlers) getOverview(w http.ResponseWriter, r *http.Request) {
	var queryCustomerID *uuid.UUID
	if raw := strings.TrimSpace(r.URL.Query().Get("customer_id")); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid customer_id")
			return
		}
		queryCustomerID = &id
	}
	customerID, err := h.ResolveCustomerID(r, queryCustomerID)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	if customerID == uuid.Nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "customer_id required")
		return
	}

	includeBalance := false
	includeLicense := false
	if snap, ok := h.SnapshotFromRequest(r); ok {
		includeBalance = snap.Has(authz.PermBillingRead) || snap.Has("customers:read")
		includeLicense = snap.Has(authz.PermBillingRead) || snap.Has("customers:read")
	}

	actorUserID := uuid.Nil
	if h.ActorUserID != nil {
		if id, ok := h.ActorUserID(r); ok {
			actorUserID = id
		}
	}

	out, err := h.Team.GetTeamOverview(r.Context(), customerID, includeBalance, includeLicense, actorUserID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpresponse.Error(w, http.StatusNotFound, "NOT_FOUND", "customer not found")
			return
		}
		h.writeServiceError(w, err)
		return
	}
	if out.Members == nil {
		out.Members = []TeamMemberDTO{}
	}
	httpresponse.JSON(w, http.StatusOK, out)
}

func (h *TeamHTTPHandlers) writeServiceError(w http.ResponseWriter, err error) {
	if h.WriteServiceError != nil {
		h.WriteServiceError(w, err)
		return
	}
	httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL", "internal error")
}

type TeamOverviewService struct {
	Pool            *pgxpool.Pool
	ClickHouseQuery *database.ClickHouseQuery
	ReportCHTimeout func() time.Duration
	CHIngestionLag  func(context.Context) (time.Duration, error)
}

func (s *TeamOverviewService) GetTeamOverview(ctx context.Context, customerID uuid.UUID, includeBalance, includeLicense bool, actorUserID uuid.UUID) (TeamOverviewDTO, error) {
	if s == nil || s.Pool == nil {
		return TeamOverviewDTO{}, errors.New("team service unavailable")
	}
	var out TeamOverviewDTO
	out.CustomerID = customerID.String()

	err := s.Pool.QueryRow(ctx, `SELECT name, balance, currency, COALESCE(cost_center, '') FROM customers WHERE id = $1`, customerID).
		Scan(&out.CustomerName, &out.BalanceMicro, &out.Currency, &out.CostCenter)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return TeamOverviewDTO{}, pgx.ErrNoRows
		}
		return TeamOverviewDTO{}, err
	}
	if !includeBalance {
		out.BalanceMicro = 0
		out.Currency = ""
	}

	if includeLicense {
		var state, planCode string
		var validUntil pgtype.Timestamptz
		err := s.Pool.QueryRow(ctx, `
			SELECT state, plan_code, valid_until
			FROM billing.license_status
			LIMIT 1`,
		).Scan(&state, &planCode, &validUntil)
		if err == nil {
			lic := &TeamLicenseDTO{State: state, PlanCode: planCode}
			if validUntil.Valid {
				lic.ValidUntil = validUntil.Time.UTC().Format(time.RFC3339)
			}
			out.License = lic
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return TeamOverviewDTO{}, err
		}
	}

	out.Members = []TeamMemberDTO{}
	pendingTotal, pendingForActor, countErr := countTeamBudgetApprovals(ctx, s.Pool, customerID, actorUserID)
	if countErr != nil {
		return TeamOverviewDTO{}, countErr
	}
	out.PendingApprovalsCount = pendingTotal
	out.PendingForMeCount = pendingForActor
	return out, nil
}

func (h *TeamHTTPHandlers) getTeamMetrics(w http.ResponseWriter, r *http.Request) {
	if h.Team == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "UNAVAILABLE", "team service unavailable")
		return
	}
	var queryCustomerID *uuid.UUID
	if raw := strings.TrimSpace(r.URL.Query().Get("customer_id")); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid customer_id")
			return
		}
		queryCustomerID = &id
	}
	customerID, err := h.ResolveCustomerID(r, queryCustomerID)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	if customerID == uuid.Nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "customer_id required")
		return
	}
	from, to, err := parseTeamMetricsRange(r)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	out, err := h.Team.GetTeamMetrics(r.Context(), customerID, from, to)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, out)
}

func parseTeamMetricsRange(r *http.Request) (time.Time, time.Time, error) {
	from, to, err := reports.ParseReportRange(r)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	if err := reports.ValidateChartRange(from, to); err != nil {
		return time.Time{}, time.Time{}, err
	}
	return from, to, nil
}

func (h *TeamHTTPHandlers) listMyBudgetApprovals(w http.ResponseWriter, r *http.Request) {
	if h.Pool == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "UNAVAILABLE", "team service unavailable")
		return
	}
	actorUserID := uuid.Nil
	if h.ActorUserID != nil {
		if id, ok := h.ActorUserID(r); ok {
			actorUserID = id
		}
	}
	if actorUserID == uuid.Nil {
		httpresponse.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}
	customerID, err := h.ResolveCustomerID(r, nil)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	if customerID == uuid.Nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "customer_id required")
		return
	}
	if err := verifyTeamMember(r.Context(), h.Pool, customerID, actorUserID); err != nil {
		if errors.Is(err, errTeamMemberForbidden) {
			httpresponse.Error(w, http.StatusForbidden, "FORBIDDEN", "forbidden")
			return
		}
		h.writeServiceError(w, err)
		return
	}
	limit := TeamBudgetApprovalsDefaultLimit
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		parsed, parseErr := strconv.Atoi(raw)
		if parseErr != nil || parsed <= 0 {
			httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid limit")
			return
		}
		limit = parsed
	}
	offset := 0
	if raw := strings.TrimSpace(r.URL.Query().Get("offset")); raw != "" {
		parsed, parseErr := strconv.Atoi(raw)
		if parseErr != nil || parsed < 0 {
			httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid offset")
			return
		}
		offset = parsed
	}
	items, total, err := listTeamBudgetApprovalsForUser(r.Context(), h.Pool, customerID, actorUserID, limit, offset)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	if items == nil {
		items = []TeamBudgetApprovalDTO{}
	}
	httpresponse.JSON(w, http.StatusOK, TeamBudgetApprovalsListResponse{
		Items:  items,
		Total:  total,
		Limit:  normalizeTeamBudgetApprovalsLimit(limit),
		Offset: normalizeTeamBudgetApprovalsOffset(offset),
	})
}

var errTeamMemberForbidden = errors.New("team member forbidden")

func verifyTeamMember(ctx context.Context, pool *pgxpool.Pool, customerID, userID uuid.UUID) error {
	var exists bool
	err := pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM users WHERE id = $1 AND customer_id = $2
		)`, userID, customerID).Scan(&exists)
	if err != nil {
		return err
	}
	if !exists {
		return errTeamMemberForbidden
	}
	return nil
}

func (s *TeamOverviewService) ListTeamMembers(ctx context.Context, customerID uuid.UUID, limit, offset int) ([]TeamMemberDTO, int64, error) {
	if s == nil || s.Pool == nil {
		return nil, 0, errors.New("team service unavailable")
	}
	return listTeamMembers(ctx, s.Pool, customerID, limit, offset)
}

func listTeamMembers(ctx context.Context, pool *pgxpool.Pool, customerID uuid.UUID, limit, offset int) ([]TeamMemberDTO, int64, error) {
	limit = normalizeTeamMembersLimit(limit)
	offset = normalizeTeamMembersOffset(offset)

	var total int64
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM users WHERE customer_id = $1`, customerID).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := pool.Query(ctx, `
		SELECT u.id, u.email, u.role, u.team_id, u.created_at, u.is_blocked,
			COALESCE(cc.campaigns_owned, 0) AS campaigns_owned,
			COALESCE(l.spend_cap_micro, 0)
		FROM users u
		LEFT JOIN team_member_limits l ON l.user_id = u.id
		LEFT JOIN (
			SELECT owner_user_id, COUNT(*)::bigint AS campaigns_owned
			FROM campaigns
			WHERE customer_id = $1
			GROUP BY owner_user_id
		) cc ON cc.owner_user_id = u.id
		WHERE u.customer_id = $1
		ORDER BY u.email
		LIMIT $2 OFFSET $3`, customerID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	out := make([]TeamMemberDTO, 0, limit)
	for rows.Next() {
		var member TeamMemberDTO
		var userID uuid.UUID
		var teamID pgtype.UUID
		var created time.Time
		if err := rows.Scan(&userID, &member.Email, &member.Role, &teamID, &created, &member.IsBlocked, &member.CampaignsOwned, &member.SpendCapMicro); err != nil {
			return nil, 0, err
		}
		member.UserID = userID.String()
		if teamID.Valid {
			member.TeamID = uuid.UUID(teamID.Bytes).String()
		}
		member.CreatedAt = created.UTC().Format(time.RFC3339)
		member.CreatedAtDisplay = coldpath.RFC3339Display(member.CreatedAt)
		out = append(out, member)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return out, total, nil
}
