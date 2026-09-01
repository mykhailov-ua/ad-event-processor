package platformadmin

import (
	"context"
	"net/http"

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
	ApplyRateLimit  func(http.HandlerFunc) http.HandlerFunc
	RequireAuth     func(http.HandlerFunc) http.HandlerFunc
	Freshness       func(context.Context) reports.DataFreshnessDTO
	BuildNav        func(context.Context) []SessionNavItemDTO
	ResolveUser     func(context.Context) (SessionUser, bool)
	ResolveAuthUser BootstrapAuthUserResolver
	EulaSnapshot    EulaBootstrapResolver
	NormalizeRole   func(string) string
}

func (h *SessionHTTPHandlers) Register(mux *http.ServeMux) {
	if h == nil || mux == nil {
		return
	}
	limit := h.ApplyRateLimit
	auth := h.RequireAuth
	if limit == nil {
		limit = func(next http.HandlerFunc) http.HandlerFunc { return next }
	}
	if auth == nil {
		auth = func(next http.HandlerFunc) http.HandlerFunc { return next }
	}
	mux.HandleFunc("GET /api/v1/session", limit(auth(h.getSession)))
	mux.HandleFunc("GET /api/v1/session/bootstrap", limit(auth(h.getSessionBootstrap)))
}

func (h *SessionHTTPHandlers) getSession(w http.ResponseWriter, r *http.Request) {
	httpresponse.JSON(w, http.StatusOK, h.buildSessionResponse(r.Context()))
}
