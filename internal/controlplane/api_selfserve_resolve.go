package controlplane

import (
	"net/http"

	"github.com/google/uuid"
)

func (handler *Handler) selfServePerm(next http.HandlerFunc, permission string) http.HandlerFunc {
	if handler.authMiddleware != nil {
		return handler.authMiddleware.RequireSelfServe(permission)(next)
	}
	return handler.perm(next, permission)
}

func (handler *Handler) resolveSelfServeCustomerID(r *http.Request, bodyCustomerID *uuid.UUID) (uuid.UUID, error) {
	u, ok := GetUser(r.Context())
	if !ok {
		return uuid.Nil, errForbidden
	}
	if u.IsUser() {
		if bodyCustomerID != nil && *bodyCustomerID != uuid.Nil && *bodyCustomerID != u.CustomerID {
			return uuid.Nil, errForbidden
		}
		return u.CustomerID, nil
	}
	if bodyCustomerID == nil || *bodyCustomerID == uuid.Nil {
		return uuid.Nil, errValidation("customer_id is required")
	}
	return *bodyCustomerID, nil
}
