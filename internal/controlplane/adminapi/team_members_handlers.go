package adminapi

import (
	"context"
	"net/http"

	"github.com/bidshard/ad-event-processor/pkg/coldpath"
	"github.com/bidshard/ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
)

// TeamGovernance covers CPA-M5 team lead mutations (invite, caps, approvals).
type TeamGovernance interface {
	InviteTeamMember(ctx context.Context, customerID uuid.UUID, email, role string) (TeamMemberDTO, error)
	UpdateTeamMember(ctx context.Context, customerID, userID uuid.UUID, req UpdateTeamMemberRequest) (TeamMemberDTO, error)
	ListTeamBudgetApprovals(ctx context.Context, customerID uuid.UUID) ([]TeamBudgetApprovalDTO, error)
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
	mux.HandleFunc("POST /api/v1/team/members", limit(teamWrite(h.inviteMember)))
	mux.HandleFunc("PATCH /api/v1/team/members/{id}", limit(teamWrite(h.patchMember)))
	mux.HandleFunc("GET /api/v1/team/budget-approvals", limit(perm(
		[]string{"campaigns:read", "billing:read"},
		h.listBudgetApprovals,
	)))
	mux.HandleFunc("POST /api/v1/team/budget-approvals/{id}/approve", limit(teamWrite(h.approveBudget)))
	mux.HandleFunc("POST /api/v1/team/budget-approvals/{id}/deny", limit(teamWrite(h.denyBudget)))
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
	member, err := h.Governance.InviteTeamMember(r.Context(), customerID, req.Email, req.Role)
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
	items, err := h.Governance.ListTeamBudgetApprovals(r.Context(), customerID)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	if items == nil {
		items = []TeamBudgetApprovalDTO{}
	}
	httpresponse.JSON(w, http.StatusOK, items)
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
