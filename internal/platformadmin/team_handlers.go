package platformadmin

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"ad-event-processor/internal/controlplane/authz"
	"ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TeamMemberDTO struct {
	UserID         string `json:"user_id"`
	Email          string `json:"email"`
	Role           string `json:"role"`
	CampaignsOwned int64  `json:"campaigns_owned"`
	CreatedAt      string `json:"created_at,omitempty"`
	IsBlocked      bool   `json:"is_blocked,omitempty"`
	SpendCapMicro  int64  `json:"spend_cap_micro,omitempty"`
}

type TeamBudgetApprovalDTO struct {
	ID                   string `json:"id"`
	UserID               string `json:"user_id"`
	CampaignID           string `json:"campaign_id"`
	RequestedBudgetMicro int64  `json:"requested_budget_micro"`
	PreviousBudgetMicro  int64  `json:"previous_budget_micro"`
	Status               string `json:"status"`
	CreatedAt            string `json:"created_at,omitempty"`
}

type InviteTeamMemberRequest struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}

type UpdateTeamMemberRequest struct {
	Role          *string `json:"role,omitempty"`
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
	CustomerID   string          `json:"customer_id"`
	CustomerName string          `json:"customer_name"`
	CostCenter   string          `json:"cost_center,omitempty"`
	BalanceMicro int64           `json:"balance_micro,omitempty"`
	Currency     string          `json:"currency,omitempty"`
	License      *TeamLicenseDTO `json:"license,omitempty"`
	Members      []TeamMemberDTO `json:"members"`
}

type TeamOverviewReader interface {
	GetTeamOverview(ctx context.Context, customerID uuid.UUID, includeBalance, includeLicense bool) (TeamOverviewDTO, error)
}

type TeamHTTPHandlers struct {
	Team                 TeamOverviewReader
	Governance           TeamGovernance
	ApplyRateLimit       func(http.HandlerFunc) http.HandlerFunc
	RequireAnyPermission func([]string, http.HandlerFunc) http.HandlerFunc
	RequireTeamWrite     func(http.HandlerFunc) http.HandlerFunc
	ResolveCustomerID    func(*http.Request, *uuid.UUID) (uuid.UUID, error)
	SnapshotFromRequest  func(*http.Request) (authz.Snapshot, bool)
	ActorUserID          func(*http.Request) (uuid.UUID, bool)
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
		[]string{"campaigns:read", "billing:read"},
		h.getOverview,
	)))
	h.registerTeamGovernanceRoutes(mux, limit, perm)
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

	out, err := h.Team.GetTeamOverview(r.Context(), customerID, includeBalance, includeLicense)
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
	Pool *pgxpool.Pool
}

func (s *TeamOverviewService) GetTeamOverview(ctx context.Context, customerID uuid.UUID, includeBalance, includeLicense bool) (TeamOverviewDTO, error) {
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

	rows, err := s.Pool.Query(ctx, `
		SELECT u.id, u.email, u.role, u.created_at, u.is_blocked,
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
		ORDER BY u.email`, customerID)
	if err != nil {
		return TeamOverviewDTO{}, err
	}
	defer rows.Close()

	for rows.Next() {
		var member TeamMemberDTO
		var userID uuid.UUID
		var created time.Time
		if err := rows.Scan(&userID, &member.Email, &member.Role, &created, &member.IsBlocked, &member.CampaignsOwned, &member.SpendCapMicro); err != nil {
			return TeamOverviewDTO{}, err
		}
		member.UserID = userID.String()
		member.CreatedAt = created.UTC().Format(time.RFC3339)
		out.Members = append(out.Members, member)
	}
	return out, rows.Err()
}
