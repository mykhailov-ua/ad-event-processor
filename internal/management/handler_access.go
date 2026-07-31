package management

import (
	"net/http"

	"github.com/google/uuid"
)

var errForbidden = &forbiddenError{}

type forbiddenError struct{}

func (e *forbiddenError) Error() string { return "forbidden" }

func (h *Handler) ensureCampaignAccess(r *http.Request, campaignID uuid.UUID) error {
	u, ok := GetUser(r.Context())
	if !ok || !u.IsUser() {
		return nil
	}
	camp, err := h.svc.GetCampaign(r.Context(), campaignID)
	if err != nil {
		return err
	}
	if uuid.UUID(camp.CustomerID.Bytes) != u.CustomerID {
		return errForbidden
	}
	return nil
}

func (h *Handler) ensureCustomerAccess(r *http.Request, customerID string) error {
	u, ok := GetUser(r.Context())
	if !ok || !u.IsUser() {
		return nil
	}
	cid, err := uuid.Parse(customerID)
	if err != nil {
		return err
	}
	if u.CustomerID != cid {
		return errForbidden
	}
	return nil
}
