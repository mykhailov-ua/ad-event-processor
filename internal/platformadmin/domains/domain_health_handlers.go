package domains

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"strings"
	"time"

	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
)

type DomainHealthService interface {
	ListDomainHealth(ctx context.Context) ([]DomainHealthDTO, error)
	AddCustomDomain(ctx context.Context, hostname string) (DomainHealthDTO, error)
	DeleteCustomDomain(ctx context.Context, hostname string) error
	ProbeDomainNow(ctx context.Context, hostname string) (DomainHealthDTO, error)
	SetupDomainSSL(ctx context.Context, hostname string) (DomainSSLSetupResult, error)
	IsTLSAllowed(ctx context.Context, hostname string) (bool, error)
	ParkDomain(ctx context.Context, req ParkDomainRequest) (ParkDomainResponse, error)
	SetupWildcardSSL(ctx context.Context, req WildcardSSLRequest) (WildcardSSLResponse, error)
	ListCloudflareZones(ctx context.Context) ([]CloudflareZone, error)
	StartBulkParkProbe(ctx context.Context, req DomainBulkRequest) (DomainBulkJobStatus, error)
	StartBulkSSL(ctx context.Context, req DomainBulkRequest) (DomainBulkJobStatus, error)
	GetBulkJob(ctx context.Context, jobID string) (DomainBulkJobStatus, error)
	BurnDomain(ctx context.Context, hostname string, req BurnDomainRequest) (BurnDomainResponse, error)
}

type DomainHealthDTO struct {
	Hostname          string     `json:"hostname"`
	Role              string     `json:"role"`
	HealthStatus      string     `json:"health_status"`
	SSLStatus         string     `json:"ssl_status"`
	SSLNotAfter       *time.Time `json:"ssl_not_after,omitempty"`
	HTTPStatus        *int       `json:"http_status,omitempty"`
	ProbeLatencyMs    *int       `json:"probe_latency_ms,omitempty"`
	ProbeDetail       string     `json:"probe_detail,omitempty"`
	LastProbeAt       *time.Time `json:"last_probe_at,omitempty"`
	UpdatedAt         time.Time  `json:"updated_at"`
	AcmeState         string     `json:"acme_state,omitempty"`
	CloudflareProxied bool       `json:"cloudflare_proxied,omitempty"`
	WildcardZone      string     `json:"wildcard_zone,omitempty"`
	PoolID            string     `json:"pool_id,omitempty"`
	PoolStatus        string     `json:"pool_status,omitempty"`
}

type DomainSSLSetupResult struct {
	Hostname string `json:"hostname"`
	Status   string `json:"status"`
	Message  string `json:"message"`
	Output   string `json:"output,omitempty"`
}

type DomainTLSAllowedResponse struct {
	Allowed bool `json:"allowed"`
}

type ParkDomainRequest struct {
	Domain           string     `json:"domain"`
	CloudflareZoneID string     `json:"cloudflare_zone_id"`
	PoolID           *uuid.UUID `json:"pool_id,omitempty"`
}

type ParkDomainResponse struct {
	Success     bool      `json:"success"`
	DNSRecordID string    `json:"dns_record_id"`
	SSLStatus   string    `json:"ssl_status"`
	Hostname    string    `json:"hostname,omitempty"`
	PoolID      uuid.UUID `json:"pool_id,omitempty"`
}

type DomainHealthHTTPHandlers struct {
	Service           DomainHealthService
	ApplyRateLimit    func(http.HandlerFunc) http.HandlerFunc
	RequirePermission func(string, http.HandlerFunc) http.HandlerFunc
	TLSAskToken       string
	TLSAskAllowLocal  bool
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
	mux.HandleFunc("POST /api/v1/domains/park", limit(perm("settings:write", h.parkDomain)))
	mux.HandleFunc("DELETE /api/v1/domains/{hostname}", limit(perm("settings:write", h.deleteDomain)))
	mux.HandleFunc("POST /api/v1/domains/{hostname}/probe", limit(perm("settings:write", h.probeDomain)))
	mux.HandleFunc("POST /api/v1/domains/{hostname}/ssl/setup", limit(perm("settings:write", h.setupSSL)))
	mux.HandleFunc("GET /api/v1/domains/cloudflare/zones", limit(perm("settings:read", h.listCloudflareZones)))
	mux.HandleFunc("POST /api/v1/ops/domains/wildcard-ssl", limit(perm("settings:write", h.setupWildcardSSL)))
	mux.HandleFunc("POST /api/v1/ops/domains/bulk", limit(perm("settings:write", h.bulkParkProbe)))
	mux.HandleFunc("POST /api/v1/ops/domains/bulk-ssl", limit(perm("settings:write", h.bulkSSL)))
	mux.HandleFunc("GET /api/v1/ops/domains/jobs/{job_id}", limit(perm("settings:read", h.getBulkJob)))
	mux.HandleFunc("PATCH /api/v1/ops/domains/{hostname}/burn", limit(perm("settings:write", h.burnDomain)))

	mux.HandleFunc("GET /api/v1/ops/domains/tls-allowed", limit(h.tlsAllowed))
	mux.HandleFunc("GET /api/v1/ops/domains/{hostname}/tls-allowed", limit(h.tlsAllowed))
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

func (h *DomainHealthHTTPHandlers) parkDomain(w http.ResponseWriter, r *http.Request) {
	req, ok := coldpath.DecodeRequestOrBadRequest[ParkDomainRequest](w, r, coldpath.DefaultMaxBody)
	if !ok {
		return
	}
	result, err := h.Service.ParkDomain(r.Context(), req)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	httpresponse.JSON(w, http.StatusOK, result)
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

func (h *DomainHealthHTTPHandlers) listCloudflareZones(w http.ResponseWriter, r *http.Request) {
	zones, err := h.Service.ListCloudflareZones(r.Context())
	if err != nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "UNAVAILABLE", err.Error())
		return
	}
	if zones == nil {
		zones = []CloudflareZone{}
	}
	httpresponse.JSON(w, http.StatusOK, zones)
}

func (h *DomainHealthHTTPHandlers) setupWildcardSSL(w http.ResponseWriter, r *http.Request) {
	req, ok := coldpath.DecodeRequestOrBadRequest[WildcardSSLRequest](w, r, coldpath.DefaultMaxBody)
	if !ok {
		return
	}
	result, err := h.Service.SetupWildcardSSL(r.Context(), req)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	status := http.StatusOK
	if result.AcmeState == "failed" {
		status = http.StatusBadGateway
	}
	httpresponse.JSON(w, status, result)
}

func (h *DomainHealthHTTPHandlers) bulkParkProbe(w http.ResponseWriter, r *http.Request) {
	req, ok := coldpath.DecodeRequestOrBadRequest[DomainBulkRequest](w, r, coldpath.DefaultMaxBody)
	if !ok {
		return
	}
	status, err := h.Service.StartBulkParkProbe(r.Context(), req)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	httpresponse.JSON(w, http.StatusAccepted, status)
}

func (h *DomainHealthHTTPHandlers) bulkSSL(w http.ResponseWriter, r *http.Request) {
	req, ok := coldpath.DecodeRequestOrBadRequest[DomainBulkRequest](w, r, coldpath.DefaultMaxBody)
	if !ok {
		return
	}
	status, err := h.Service.StartBulkSSL(r.Context(), req)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	httpresponse.JSON(w, http.StatusAccepted, status)
}

func (h *DomainHealthHTTPHandlers) getBulkJob(w http.ResponseWriter, r *http.Request) {
	jobID := r.PathValue("job_id")
	status, err := h.Service.GetBulkJob(r.Context(), jobID)
	if err != nil {
		if err.Error() == "job not found" {
			httpresponse.Error(w, http.StatusNotFound, "NOT_FOUND", err.Error())
			return
		}
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	httpresponse.JSON(w, http.StatusOK, status)
}

func (h *DomainHealthHTTPHandlers) burnDomain(w http.ResponseWriter, r *http.Request) {
	host := r.PathValue("hostname")
	var req BurnDomainRequest
	if r.ContentLength > 0 {
		body, err := coldpath.ReadLimitedBody(w, r, coldpath.DefaultMaxBody)
		if err != nil {
			return
		}
		if len(body) > 0 {
			if err := json.Unmarshal(body, &req); err != nil {
				httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
				return
			}
		}
	}
	result, err := h.Service.BurnDomain(r.Context(), host, req)
	if err != nil {
		if err.Error() == "domain not found in pool" {
			httpresponse.Error(w, http.StatusNotFound, "NOT_FOUND", err.Error())
			return
		}
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	httpresponse.JSON(w, http.StatusOK, result)
}

func (h *DomainHealthHTTPHandlers) tlsAllowed(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpresponse.Error(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "GET only")
		return
	}
	if !h.authorizeTLSAsk(r) {
		httpresponse.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid tls ask credentials")
		return
	}
	host := tlsAskHostname(r)
	if host == "" {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "hostname required")
		return
	}
	allowed, err := h.Service.IsTLSAllowed(r.Context(), host)
	if err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	if !allowed {
		httpresponse.Error(w, http.StatusForbidden, "FORBIDDEN", "tls not allowed for hostname")
		return
	}
	httpresponse.JSON(w, http.StatusOK, DomainTLSAllowedResponse{Allowed: true})
}

func tlsAskHostname(r *http.Request) string {
	host := strings.TrimSpace(r.PathValue("hostname"))
	if host != "" {
		return host
	}
	return strings.TrimSpace(r.URL.Query().Get("domain"))
}

func (h *DomainHealthHTTPHandlers) authorizeTLSAsk(r *http.Request) bool {
	if h == nil {
		return false
	}
	if token := strings.TrimSpace(h.TLSAskToken); token != "" {
		got := strings.TrimSpace(r.Header.Get("X-Caddy-Ask-Token"))
		if got == "" {
			if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
				got = strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
			}
		}
		if got == "" {
			got = strings.TrimSpace(r.URL.Query().Get("token"))
		}
		return got != "" && got == token
	}
	if !h.TLSAskAllowLocal {
		return false
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
