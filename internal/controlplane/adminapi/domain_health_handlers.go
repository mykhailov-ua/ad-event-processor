package adminapi

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/bidshard/ad-event-processor/pkg/coldpath"
	"github.com/bidshard/ad-event-processor/pkg/httpresponse"
)

type DomainHealthService interface {
	ListDomainHealth(ctx context.Context) ([]DomainHealthDTO, error)
	AddCustomDomain(ctx context.Context, hostname string) (DomainHealthDTO, error)
	DeleteCustomDomain(ctx context.Context, hostname string) error
	ProbeDomainNow(ctx context.Context, hostname string) (DomainHealthDTO, error)
	SetupDomainSSL(ctx context.Context, hostname string) (DomainSSLSetupResult, error)
}

type DomainHealthDTO struct {
	Hostname       string     `json:"hostname"`
	Role           string     `json:"role"`
	HealthStatus   string     `json:"health_status"`
	SSLStatus      string     `json:"ssl_status"`
	SSLNotAfter    *time.Time `json:"ssl_not_after,omitempty"`
	HTTPStatus     *int       `json:"http_status,omitempty"`
	ProbeLatencyMs *int       `json:"probe_latency_ms,omitempty"`
	ProbeDetail    string     `json:"probe_detail,omitempty"`
	LastProbeAt    *time.Time `json:"last_probe_at,omitempty"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type DomainSSLSetupResult struct {
	Hostname string `json:"hostname"`
	Status   string `json:"status"`
	Message  string `json:"message"`
	Output   string `json:"output,omitempty"`
}

type DomainHealthHTTPHandlers struct {
	Service           DomainHealthService
	ApplyRateLimit    func(http.HandlerFunc) http.HandlerFunc
	RequirePermission func(string, http.HandlerFunc) http.HandlerFunc
}

func (h *DomainHealthHTTPHandlers) Register(mux *http.ServeMux) {
	if h == nil || h.Service == nil {
		return
	}
	limit := h.ApplyRateLimit
	perm := h.RequirePermission
	if limit == nil {
		limit = func(next http.HandlerFunc) http.HandlerFunc { return next }
	}
	if perm == nil {
		perm = func(_ string, next http.HandlerFunc) http.HandlerFunc { return next }
	}
	mux.HandleFunc("GET /api/v1/domains", limit(perm("settings:read", h.listDomains)))
	mux.HandleFunc("POST /api/v1/domains", limit(perm("settings:write", h.addDomain)))
	mux.HandleFunc("DELETE /api/v1/domains/{hostname}", limit(perm("settings:write", h.deleteDomain)))
	mux.HandleFunc("POST /api/v1/domains/{hostname}/probe", limit(perm("settings:write", h.probeDomain)))
	mux.HandleFunc("POST /api/v1/domains/{hostname}/ssl/setup", limit(perm("settings:write", h.setupSSL)))
}

func (h *DomainHealthHTTPHandlers) listDomains(w http.ResponseWriter, r *http.Request) {
	domains, err := h.Service.ListDomainHealth(r.Context())
	if err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	if domains == nil {
		domains = []DomainHealthDTO{}
	}
	httpresponse.JSON(w, http.StatusOK, domains)
}

func (h *DomainHealthHTTPHandlers) addDomain(w http.ResponseWriter, r *http.Request) {
	body, err := coldpath.ReadLimitedBody(w, r, coldpath.DefaultMaxBody)
	if err != nil {
		return
	}
	var req struct {
		Hostname string `json:"hostname"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}
	dto, err := h.Service.AddCustomDomain(r.Context(), req.Hostname)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	httpresponse.JSON(w, http.StatusCreated, dto)
}

func (h *DomainHealthHTTPHandlers) deleteDomain(w http.ResponseWriter, r *http.Request) {
	host := r.PathValue("hostname")
	if err := h.Service.DeleteCustomDomain(r.Context(), host); err != nil {
		if err.Error() == "domain not found or not removable" {
			httpresponse.Error(w, http.StatusNotFound, "NOT_FOUND", err.Error())
			return
		}
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *DomainHealthHTTPHandlers) probeDomain(w http.ResponseWriter, r *http.Request) {
	host := r.PathValue("hostname")
	dto, err := h.Service.ProbeDomainNow(r.Context(), host)
	if err != nil {
		if err.Error() == "domain not found" {
			httpresponse.Error(w, http.StatusNotFound, "NOT_FOUND", err.Error())
			return
		}
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	httpresponse.JSON(w, http.StatusOK, dto)
}

func (h *DomainHealthHTTPHandlers) setupSSL(w http.ResponseWriter, r *http.Request) {
	host := r.PathValue("hostname")
	result, err := h.Service.SetupDomainSSL(r.Context(), host)
	if err != nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "UNAVAILABLE", err.Error())
		return
	}
	status := http.StatusOK
	if result.Status == "failed" {
		status = http.StatusBadGateway
	}
	httpresponse.JSON(w, status, result)
}
