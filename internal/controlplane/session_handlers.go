package controlplane

import (
	"context"
	"net/http"

	"ad-event-processor/internal/controlplane/authz"
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

type SessionHTTPHandlers struct {
	ApplyRateLimit func(http.HandlerFunc) http.HandlerFunc
	Freshness      func(context.Context) DataFreshnessDTO
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
		NavItems: buildSessionNav(r.Context()),
	}
	if snap, ok := authz.SnapshotFromContext(r.Context()); ok {
		resp.MaskLevel = string(snap.Mask)
	}
	if u, ok := GetUser(r.Context()); ok {
		resp.Role = NormalizeRole(u.Role)
		if u.HasBoundCustomer() {
			resp.DefaultCustomerID = u.CustomerID.String()
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

func buildSessionNav(ctx context.Context) []SessionNavItemDTO {
	catalogRows := filterReportCatalog(ctx, reportCatalogEntries)
	items := []SessionNavItemDTO{
		{Href: "/campaigns", Label: "Campaigns", IconKey: "campaigns"},
		{Href: "/dashboards/buyer", Label: "Dashboard", IconKey: "dashboard"},
		{Href: "/reports", Label: "Reports", IconKey: "reports"},
	}
	snap, ok := authz.SnapshotFromContext(ctx)
	if ok && snap.Has("customers:read") && snap.Mask == authz.MaskFull {
		items = append(items, SessionNavItemDTO{Href: "/customers", Label: "Customers", IconKey: "customers"})
	}
	if ok && snap.Has("audit:read") {
		items = append(items, SessionNavItemDTO{Href: "/ops", Label: "Operations", IconKey: "ops"})
	}
	for _, row := range catalogRows {
		if row.Category != "fraud" {
			continue
		}
		items = append(items, SessionNavItemDTO{
			Href:    "/reports/" + row.Key,
			Label:   row.Title,
			IconKey: "report",
		})
	}
	return items
}
