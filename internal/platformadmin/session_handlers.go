package platformadmin

import (
	"context"
	"net/http"

	"ad-event-processor/internal/controlplane/authz"
	"ad-event-processor/internal/reports"
	"ad-event-processor/pkg/httpresponse"
)

type SessionNavItemDTO struct {
	Href    string `json:"href"`
	Label   string `json:"label"`
	IconKey string `json:"icon_key,omitempty"`
}

type SessionResponseDTO struct {
	Role              string              `json:"role"`
	MaskLevel         string              `json:"mask_level"`
	DefaultCustomerID string              `json:"default_customer_id,omitempty"`
	Timezone          string              `json:"timezone,omitempty"`
	StaleBanner       string              `json:"stale_banner,omitempty"`
	NavItems          []SessionNavItemDTO `json:"nav_items"`
}

type SessionUser struct {
	Role              string
	DefaultCustomerID string
}

type SessionHTTPHandlers struct {
	ApplyRateLimit func(http.HandlerFunc) http.HandlerFunc
	Freshness      func(context.Context) reports.DataFreshnessDTO
	BuildNav       func(context.Context) []SessionNavItemDTO
	ResolveUser    func(context.Context) (SessionUser, bool)
	NormalizeRole  func(string) string
}

func (h *SessionHTTPHandlers) Register(mux *http.ServeMux) {
	if h == nil || mux == nil {
		return
	}
	limit := h.ApplyRateLimit
	if limit == nil {
		limit = func(next http.HandlerFunc) http.HandlerFunc { return next }
	}
	mux.HandleFunc("GET /api/v1/session", limit(h.getSession))
}

func (h *SessionHTTPHandlers) getSession(w http.ResponseWriter, r *http.Request) {
	resp := SessionResponseDTO{
		Timezone: "UTC",
	}
	if h.BuildNav != nil {
		resp.NavItems = h.BuildNav(r.Context())
	}
	if snap, ok := authz.SnapshotFromContext(r.Context()); ok {
		resp.MaskLevel = string(snap.Mask)
	}
	if h.ResolveUser != nil {
		if u, ok := h.ResolveUser(r.Context()); ok {
			role := u.Role
			if h.NormalizeRole != nil {
				role = h.NormalizeRole(role)
			}
			resp.Role = role
			resp.DefaultCustomerID = u.DefaultCustomerID
		}
	}
	if h.Freshness != nil {
		fresh := h.Freshness(r.Context())
		if fresh.Stale {
			resp.StaleBanner = "Analytics may be delayed while ClickHouse catches up."
		}
	}
	httpresponse.JSON(w, http.StatusOK, resp)
}
