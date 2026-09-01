package platformadmin

import (
	"context"
	"net/http"

	"ad-event-processor/internal/controlplane/authz"
	"ad-event-processor/pkg/httpresponse"
)

type BootstrapUserDTO struct {
	ID          string   `json:"id"`
	Email       string   `json:"email,omitempty"`
	Role        string   `json:"role"`
	CustomerID  string   `json:"customer_id"`
	Permissions []string `json:"permissions,omitempty"`
}

type EulaBootstrapDTO struct {
	EulaRequired bool   `json:"eula_required"`
	EulaAccepted bool   `json:"eula_accepted"`
	EulaVersion  string `json:"eula_version,omitempty"`
}

type SessionBootstrapDTO struct {
	User         BootstrapUserDTO   `json:"user"`
	Session      SessionResponseDTO `json:"session"`
	EulaRequired bool               `json:"eula_required"`
	EulaAccepted bool               `json:"eula_accepted"`
	EulaVersion  string             `json:"eula_version,omitempty"`
}

type BootstrapAuthUserResolver func(context.Context) (BootstrapUserDTO, bool)

type EulaBootstrapResolver func(context.Context) (EulaBootstrapDTO, error)

func (h *SessionHTTPHandlers) buildSessionResponse(ctx context.Context) SessionResponseDTO {
	resp := SessionResponseDTO{
		Timezone: "UTC",
	}
	if h.BuildNav != nil {
		resp.NavItems = h.BuildNav(ctx)
	}
	if snap, ok := authz.SnapshotFromContext(ctx); ok {
		resp.MaskLevel = string(snap.Mask)
	}
	if h.ResolveUser != nil {
		if u, ok := h.ResolveUser(ctx); ok {
			role := u.Role
			if h.NormalizeRole != nil {
				role = h.NormalizeRole(role)
			}
			resp.Role = role
			resp.DefaultCustomerID = u.DefaultCustomerID
		}
	}
	if h.Freshness != nil {
		fresh := h.Freshness(ctx)
		if fresh.Stale {
			resp.StaleBanner = "Analytics may be delayed while ClickHouse catches up."
		}
	}
	return resp
}

func (h *SessionHTTPHandlers) getSessionBootstrap(w http.ResponseWriter, r *http.Request) {
	if h.ResolveAuthUser == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "UNAVAILABLE", "session bootstrap not configured")
		return
	}
	user, ok := h.ResolveAuthUser(r.Context())
	if !ok {
		httpresponse.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized")
		return
	}

	resp := SessionBootstrapDTO{
		User:    user,
		Session: h.buildSessionResponse(r.Context()),
	}
	if h.EulaSnapshot != nil {
		eula, err := h.EulaSnapshot(r.Context())
		if err != nil {
			httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to load eula status")
			return
		}
		resp.EulaRequired = eula.EulaRequired
		resp.EulaAccepted = eula.EulaAccepted
		resp.EulaVersion = eula.EulaVersion
	}
	httpresponse.JSON(w, http.StatusOK, resp)
}
