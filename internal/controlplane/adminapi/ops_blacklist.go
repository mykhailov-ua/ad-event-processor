package adminapi

import (
	"context"
	"net/http"
	"strconv"

	"espx/pkg/coldpath"
	"espx/pkg/httpresponse"
)

type BlacklistAdmin interface {
	BlockIPWithTTL(ctx context.Context, ip, source string, ttlSeconds *int64) error
	PreviewBlockIP(ctx context.Context, ip, source string, ttlSeconds *int64) (any, error)
	UnblockIP(ctx context.Context, ip, source string) error
	ListBlacklist(ctx context.Context, limit, offset int32) (any, int64, error)
}

func (h *OpsHTTPHandlers) registerBlacklistRoutes(mux *http.ServeMux) {
	if h.Blacklist == nil {
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
	mux.HandleFunc("POST /api/v1/ops/blacklist", limit(perm("blacklist:write", h.blockIP)))
	mux.HandleFunc("DELETE /api/v1/ops/blacklist", limit(perm("blacklist:write", h.unblockIP)))
	mux.HandleFunc("GET /api/v1/ops/blacklist", limit(perm("blacklist:read", h.listBlacklist)))
}

func (h *OpsHTTPHandlers) blockIP(w http.ResponseWriter, r *http.Request) {
	req, err := coldpath.DecodeRequest[struct {
		IP         string `json:"ip"`
		Source     string `json:"source"`
		TTLSeconds *int64 `json:"ttl_seconds,omitempty"`
	}](w, r, coldpath.DefaultMaxBody)
	if err != nil || req.IP == "" {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}
	if r.Header.Get("X-Dry-Run") == "1" || r.URL.Query().Get("dry_run") == "1" {
		preview, err := h.Blacklist.PreviewBlockIP(r.Context(), req.IP, req.Source, req.TTLSeconds)
		if err != nil {
			h.writeServiceError(w, err)
			return
		}
		httpresponse.JSON(w, http.StatusOK, preview)
		return
	}
	if err := h.Blacklist.BlockIPWithTTL(r.Context(), req.IP, req.Source, req.TTLSeconds); err != nil {
		h.writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (h *OpsHTTPHandlers) unblockIP(w http.ResponseWriter, r *http.Request) {
	req, err := coldpath.DecodeRequest[struct {
		IP     string `json:"ip"`
		Source string `json:"source"`
	}](w, r, coldpath.DefaultMaxBody)
	if err != nil || req.IP == "" {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}
	if err := h.Blacklist.UnblockIP(r.Context(), req.IP, req.Source); err != nil {
		h.writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *OpsHTTPHandlers) listBlacklist(w http.ResponseWriter, r *http.Request) {
	limit, offset := parseAPIPagination(r)
	items, total, err := h.Blacklist.ListBlacklist(r.Context(), limit, offset)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	w.Header().Set("X-Total-Count", strconv.FormatInt(total, 10))
	httpresponse.JSON(w, http.StatusOK, items)
}
