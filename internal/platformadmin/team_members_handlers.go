package platformadmin

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
)

type TeamGovernance interface {
	InviteTeamMember(ctx context.Context, customerID uuid.UUID, email, role string) (TeamMemberDTO, error)
	UpdateTeamMember(ctx context.Context, customerID, userID uuid.UUID, req UpdateTeamMemberRequest) (TeamMemberDTO, error)
	ListTeamBudgetApprovals(ctx context.Context, customerID uuid.UUID, limit, offset int) ([]TeamBudgetApprovalDTO, int64, error)
	ResolveTeamBudgetApproval(ctx context.Context, customerID, approvalID, resolverID uuid.UUID, approve bool) error
}

func (h *TeamHTTPHandlers) registerTeamGovernanceRoutes(mux *http.ServeMux, limit func(http.HandlerFunc) http.HandlerFunc, perm func([]string, http.HandlerFunc) http.HandlerFunc) {
	if h == nil || h.Governance == nil {
		return
	}
	teamWrite := h.RequireTeamWrite
	if teamWrite == nil {
		teamWrite = func(next http.HandlerFunc) http.HandlerFunc { return next }
	}
	mux.HandleFunc("GET /api/v1/team/members", limit(perm(
		[]string{"campaigns:read", "billing:read"},
		h.listMembers,
	)))
	mux.HandleFunc("POST /api/v1/team/members", limit(teamWrite(h.inviteMember)))
	mux.HandleFunc("PATCH /api/v1/team/members/{id}", limit(teamWrite(h.patchMember)))
	mux.HandleFunc("GET /api/v1/team/budget-approvals", limit(perm(
		[]string{"campaigns:read", "billing:read"},
		h.listBudgetApprovals,
	)))
	mux.HandleFunc("POST /api/v1/team/budget-approvals/{id}/approve", limit(teamWrite(h.approveBudget)))
	mux.HandleFunc("POST /api/v1/team/budget-approvals/{id}/deny", limit(teamWrite(h.denyBudget)))
}

func (h *TeamHTTPHandlers) listMembers(w http.ResponseWriter, r *http.Request) {
	if h.Team == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "UNAVAILABLE", "team service unavailable")
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
	limit := TeamMembersDefaultLimit
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid limit")
			return
		}
		limit = parsed
	}
	offset := 0
	if raw := strings.TrimSpace(r.URL.Query().Get("offset")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 0 {
			httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid offset")
			return
		}
		offset = parsed
	}

	items, total, err := h.Team.ListTeamMembers(r.Context(), customerID, limit, offset)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	if items == nil {
		items = []TeamMemberDTO{}
	}
	httpresponse.JSON(w, http.StatusOK, TeamMembersListResponse{
		Items:  items,
		Total:  total,
		Limit:  normalizeTeamMembersLimit(limit),
		Offset: normalizeTeamMembersOffset(offset),
	})
}

func (h *TeamHTTPHandlers) inviteMember(w http.ResponseWriter, r *http.Request) {
	customerID, err := h.ResolveCustomerID(r, nil)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	if customerID == uuid.Nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "customer_id required")
		return
	}
	req, ok := coldpath.DecodeRequestOrBadRequest[InviteTeamMemberRequest](w, r, coldpath.DefaultMaxBody)
	if !ok {
		return
	}
	member, err := h.Governance.InviteTeamMember(WithPanelRequest(r.Context(), r), customerID, req.Email, req.Role)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusCreated, member)
}

func (h *TeamHTTPHandlers) patchMember(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid user id")
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
	req, ok := coldpath.DecodeRequestOrBadRequest[UpdateTeamMemberRequest](w, r, coldpath.DefaultMaxBody)
	if !ok {
		return
	}
	member, err := h.Governance.UpdateTeamMember(r.Context(), customerID, userID, req)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, member)
}

func (h *TeamHTTPHandlers) listBudgetApprovals(w http.ResponseWriter, r *http.Request) {
	customerID, err := h.ResolveCustomerID(r, nil)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	if customerID == uuid.Nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "customer_id required")
		return
	}
	limit := TeamBudgetApprovalsDefaultLimit
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid limit")
			return
		}
		limit = parsed
	}
	offset := 0
	if raw := strings.TrimSpace(r.URL.Query().Get("offset")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 0 {
			httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid offset")
			return
		}
		offset = parsed
	}

	items, total, err := h.Governance.ListTeamBudgetApprovals(r.Context(), customerID, limit, offset)
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

func (h *TeamHTTPHandlers) approveBudget(w http.ResponseWriter, r *http.Request) {
	h.resolveBudgetApproval(w, r, true)
}

func (h *TeamHTTPHandlers) denyBudget(w http.ResponseWriter, r *http.Request) {
	h.resolveBudgetApproval(w, r, false)
}

func (h *TeamHTTPHandlers) resolveBudgetApproval(w http.ResponseWriter, r *http.Request, approve bool) {
	approvalID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid approval id")
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
	resolverID := uuid.Nil
	if h.ActorUserID != nil {
		if id, ok := h.ActorUserID(r); ok {
			resolverID = id
		}
	}
	if err := h.Governance.ResolveTeamBudgetApproval(r.Context(), customerID, approvalID, resolverID, approve); err != nil {
		h.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
