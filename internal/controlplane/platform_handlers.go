package controlplane

import (
	"context"
	"net/http"

	"github.com/bidshard/ad-event-processor/internal/config"
	"github.com/bidshard/ad-event-processor/pkg/coldpath"
	"github.com/bidshard/ad-event-processor/pkg/httpresponse"
	"github.com/bidshard/ad-event-processor/pkg/platformconfig"
)

type PlatformConfigService interface {
	GetPlatformConfig(ctx context.Context) (platformconfig.Config, bool, error)
	GetPlatformRestartPending(ctx context.Context) ([]string, error)
	BootstrapPlatformConfig(ctx context.Context, req platformconfig.BootstrapRequest, installToken string) error
	UpdatePlatformConfig(ctx context.Context, patch platformconfig.Patch) (platformconfig.Config, []string, error)
	ApplyPlatformConfig(ctx context.Context, installRoot string) (string, error)
}

type PlatformAuthClient interface {
	Register(ctx context.Context, adminAPIKey, email, password, role, customerID string) error
}

type PlatformHTTPHandlers struct {
	Service           PlatformConfigService
	AuthClient        PlatformAuthClient
	Cfg               *config.Config
	ApplyRateLimit    func(http.HandlerFunc) http.HandlerFunc
	RequirePermission func(string, http.HandlerFunc) http.HandlerFunc
	WriteServiceError func(http.ResponseWriter, error)
}

func (h *PlatformHTTPHandlers) Register(mux *http.ServeMux) {
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
	mux.HandleFunc("GET /api/v1/settings/platform", limit(perm("settings:read", h.getPlatform)))
	mux.HandleFunc("PATCH /api/v1/settings/platform", limit(perm("settings:write", h.patchPlatform)))
	mux.HandleFunc("POST /api/v1/settings/platform/bootstrap", limit(h.postPlatformBootstrap))
	mux.HandleFunc("POST /api/v1/settings/platform/apply", limit(perm("settings:write", h.postPlatformApply)))
}

func (h *PlatformHTTPHandlers) getPlatform(w http.ResponseWriter, r *http.Request) {
	cfg, bootstrapped, err := h.Service.GetPlatformConfig(r.Context())
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	pending, err := h.Service.GetPlatformRestartPending(r.Context())
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, platformconfig.Public(cfg, bootstrapped, pending))
}

func (h *PlatformHTTPHandlers) patchPlatform(w http.ResponseWriter, r *http.Request) {
	body, err := coldpath.ReadLimitedBody(w, r, coldpath.DefaultMaxBody)
	if err != nil {
		return
	}
	patch, err := coldpath.DecodeBody[platformconfig.Patch](body)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}
	cfg, restartRequired, err := h.Service.UpdatePlatformConfig(r.Context(), patch)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, platformconfig.Public(cfg, true, restartRequired))
}

func (h *PlatformHTTPHandlers) postPlatformBootstrap(w http.ResponseWriter, r *http.Request) {
	body, err := coldpath.ReadLimitedBody(w, r, coldpath.DefaultMaxBody)
	if err != nil {
		return
	}
	req, err := coldpath.DecodeBody[platformconfig.BootstrapRequest](body)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}
	token := r.Header.Get("X-Install-Token")
	if err := h.Service.BootstrapPlatformConfig(r.Context(), req, token); err != nil {
		h.writeServiceError(w, err)
		return
	}
	if req.AdminEmail != "" && req.AdminPassword != "" && h.AuthClient != nil && h.Cfg != nil && len(h.Cfg.AdminAPIKey) > 0 {
		if err := h.AuthClient.Register(r.Context(), string(h.Cfg.AdminAPIKey), req.AdminEmail, req.AdminPassword, "A", ""); err != nil {
			h.writeServiceError(w, err)
			return
		}
	}
	cfg, _, err := h.Service.GetPlatformConfig(r.Context())
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, platformconfig.Public(cfg, true, nil))
}

type platformApplyRequest struct {
	InstallRoot string `json:"install_root,omitempty"`
}

type platformApplyResponse struct {
	WrittenPath string `json:"written_path"`
}

func (h *PlatformHTTPHandlers) postPlatformApply(w http.ResponseWriter, r *http.Request) {
	installRoot := ""
	body, err := coldpath.ReadLimitedBody(w, r, coldpath.DefaultMaxBody)
	if err != nil {
		return
	}
	if len(body) > 0 {
		req, decodeErr := coldpath.DecodeBody[platformApplyRequest](body)
		if decodeErr != nil {
			httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
			return
		}
		installRoot = req.InstallRoot
	}
	path, err := h.Service.ApplyPlatformConfig(r.Context(), installRoot)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, platformApplyResponse{WrittenPath: path})
}

func (h *PlatformHTTPHandlers) writeServiceError(w http.ResponseWriter, err error) {
	if h.WriteServiceError != nil {
		h.WriteServiceError(w, err)
		return
	}
	httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL", "request failed")
}
