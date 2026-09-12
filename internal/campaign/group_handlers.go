package campaign

import (
	"encoding/json"
	"net/http"
	"strings"

	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
)

func (h *CampaignsHTTPHandlers) registerCampaignGroupRoutes(mux *http.ServeMux, limit func(http.HandlerFunc) http.HandlerFunc, perm func([]string, http.HandlerFunc) http.HandlerFunc) {
	mux.HandleFunc("GET /api/v1/campaign-groups", limit(perm([]string{"campaigns:read", "campaigns:read:masked"}, h.listCampaignGroups)))
	mux.HandleFunc("POST /api/v1/campaign-groups", limit(perm([]string{"campaigns:write"}, h.createCampaignGroup)))
	mux.HandleFunc("GET /api/v1/campaign-groups/{id}", limit(perm([]string{"campaigns:read", "campaigns:read:masked"}, h.getCampaignGroup)))
	mux.HandleFunc("PATCH /api/v1/campaign-groups/{id}", limit(perm([]string{"campaigns:write"}, h.patchCampaignGroup)))
	mux.HandleFunc("DELETE /api/v1/campaign-groups/{id}", limit(perm([]string{"campaigns:write"}, h.deleteCampaignGroup)))
	mux.HandleFunc("POST /api/v1/campaign-groups/{id}/assign-campaigns", limit(perm([]string{"campaigns:write"}, h.assignCampaignGroupMembers)))
	mux.HandleFunc("POST /api/v1/campaign-groups/{id}/unassign-campaigns", limit(perm([]string{"campaigns:write"}, h.unassignCampaignGroupMembers)))
}

func (h *CampaignsHTTPHandlers) listCampaignGroups(w http.ResponseWriter, r *http.Request) {
	custStr := strings.TrimSpace(r.URL.Query().Get("customer_id"))
	if custStr == "" {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "customer_id required")
		return
	}
	customerID, err := uuid.Parse(custStr)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid customer_id")
		return
	}
	if h.ResolveCustomerID != nil {
		customerID, err = h.ResolveCustomerID(r, &customerID)
		if err != nil {
			h.WriteHandlerError(w, err)
			return
		}
	}
	rows, err := ListCampaignGroups(r.Context(), h.PostgresPool, customerID)
	if err != nil {
		h.WriteHandlerError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, rows)
}

func (h *CampaignsHTTPHandlers) getCampaignGroup(w http.ResponseWriter, r *http.Request) {
	groupID, err := coldpath.ParsePathUUID(r, "id")
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid campaign group id")
		return
	}
	row, err := GetCampaignGroup(r.Context(), h.PostgresPool, groupID)
	if err != nil {
		h.WriteHandlerError(w, err)
		return
	}
	if err := h.authorizeCampaignGroupCustomer(w, r, row.CustomerID); err != nil {
		return
	}
	httpresponse.JSON(w, http.StatusOK, row)
}

func (h *CampaignsHTTPHandlers) createCampaignGroup(w http.ResponseWriter, r *http.Request) {
	req, ok := coldpath.DecodeRequestOrBadRequest[CreateCampaignGroupRequest](w, r, coldpath.DefaultMaxBody)
	if !ok {
		return
	}
	customerID, err := uuid.Parse(strings.TrimSpace(req.CustomerID))
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid customer_id")
		return
	}
	if h.ResolveCustomerID != nil {
		customerID, err = h.ResolveCustomerID(r, &customerID)
		if err != nil {
			h.WriteHandlerError(w, err)
			return
		}
	}
	req.CustomerID = customerID.String()
	row, err := CreateCampaignGroup(r.Context(), h.PostgresPool, req)
	if err != nil {
		h.WriteHandlerError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusCreated, row)
}

func (h *CampaignsHTTPHandlers) patchCampaignGroup(w http.ResponseWriter, r *http.Request) {
	groupID, err := coldpath.ParsePathUUID(r, "id")
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid campaign group id")
		return
	}
	current, err := GetCampaignGroup(r.Context(), h.PostgresPool, groupID)
	if err != nil {
		h.WriteHandlerError(w, err)
		return
	}
	if err := h.authorizeCampaignGroupCustomer(w, r, current.CustomerID); err != nil {
		return
	}
	req, ok := coldpath.DecodeRequestOrBadRequest[UpdateCampaignGroupRequest](w, r, coldpath.DefaultMaxBody)
	if !ok {
		return
	}
	row, err := UpdateCampaignGroup(r.Context(), h.PostgresPool, groupID, req)
	if err != nil {
		h.WriteHandlerError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, row)
}

func (h *CampaignsHTTPHandlers) deleteCampaignGroup(w http.ResponseWriter, r *http.Request) {
	groupID, err := coldpath.ParsePathUUID(r, "id")
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid campaign group id")
		return
	}
	current, err := GetCampaignGroup(r.Context(), h.PostgresPool, groupID)
	if err != nil {
		h.WriteHandlerError(w, err)
		return
	}
	if err := h.authorizeCampaignGroupCustomer(w, r, current.CustomerID); err != nil {
		return
	}
	if err := DeleteCampaignGroup(r.Context(), h.PostgresPool, groupID); err != nil {
		h.WriteHandlerError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *CampaignsHTTPHandlers) assignCampaignGroupMembers(w http.ResponseWriter, r *http.Request) {
	h.mutateCampaignGroupMembers(w, r, true)
}

func (h *CampaignsHTTPHandlers) unassignCampaignGroupMembers(w http.ResponseWriter, r *http.Request) {
	h.mutateCampaignGroupMembers(w, r, false)
}

func (h *CampaignsHTTPHandlers) mutateCampaignGroupMembers(w http.ResponseWriter, r *http.Request, assign bool) {
	groupID, err := coldpath.ParsePathUUID(r, "id")
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid campaign group id")
		return
	}
	current, err := GetCampaignGroup(r.Context(), h.PostgresPool, groupID)
	if err != nil {
		h.WriteHandlerError(w, err)
		return
	}
	if err := h.authorizeCampaignGroupCustomer(w, r, current.CustomerID); err != nil {
		return
	}
	body, err := coldpath.ReadLimitedBody(w, r, coldpath.DefaultMaxBody)
	if err != nil {
		return
	}
	var req AssignCampaignGroupMembersRequest
	if err := json.Unmarshal(body, &req); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid json body")
		return
	}
	campaignIDs, err := ParseCampaignGroupIDs(req.CampaignIDs)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid campaign_ids")
		return
	}
	if h.AuthorizeCampaignIDsAccess != nil {
		denied := h.AuthorizeCampaignIDsAccess(r, campaignIDs)
		for _, id := range campaignIDs {
			if denied[id] != nil {
				h.WriteHandlerError(w, ErrForbidden)
				return
			}
		}
	}
	var resp AssignCampaignGroupMembersResponse
	if assign {
		resp, err = AssignCampaignsToGroup(r.Context(), h.PostgresPool, groupID, campaignIDs)
	} else {
		resp, err = UnassignCampaignsFromGroup(r.Context(), h.PostgresPool, groupID, campaignIDs)
	}
	if err != nil {
		h.WriteHandlerError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, resp)
}

func (h *CampaignsHTTPHandlers) authorizeCampaignGroupCustomer(w http.ResponseWriter, r *http.Request, customerID string) error {
	if h.ResolveCustomerID == nil {
		return nil
	}
	id, err := uuid.Parse(customerID)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid customer_id")
		return err
	}
	if _, err := h.ResolveCustomerID(r, &id); err != nil {
		h.WriteHandlerError(w, err)
		return err
	}
	return nil
}
