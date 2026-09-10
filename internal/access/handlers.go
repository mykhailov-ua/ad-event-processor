package access

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
)

type HTTPHandlers struct {
	Store                 *RolesStore
	ApplyRateLimit        func(http.HandlerFunc) http.HandlerFunc
	RequirePermission     func(string, http.HandlerFunc) http.HandlerFunc
	RequireAnyPermission  func([]string, http.HandlerFunc) http.HandlerFunc
	ActorUserID           func(*http.Request) (uuid.UUID, bool)
	WriteServiceError     func(http.ResponseWriter, error)
	Audit                 func(ctx context.Context, actorID uuid.UUID, changes, metadata any)
}

func (h *HTTPHandlers) Register(mux *http.ServeMux) {
	if h == nil || h.Store == nil {
		return
	}
	limit := h.ApplyRateLimit
	perm := h.RequirePermission
	permAny := h.RequireAnyPermission
	if limit == nil {
		limit = func(next http.HandlerFunc) http.HandlerFunc { return next }
	}
	if perm == nil {
		perm = func(_ string, next http.HandlerFunc) http.HandlerFunc { return next }
	}
	if permAny == nil {
		permAny = func(perms []string, next http.HandlerFunc) http.HandlerFunc {
			if len(perms) == 0 {
				return next
			}
			return perm(perms[0], next)
		}
	}
	mux.HandleFunc("GET /api/v1/access/catalog", limit(perm("access:read", h.getCatalog)))
	mux.HandleFunc("GET /api/v1/access/roles", limit(perm("access:read", h.getRoles)))
	mux.HandleFunc("GET /api/v1/access/roles.yaml", limit(perm("access:read", h.getRolesYAML)))
	mux.HandleFunc("PUT /api/v1/access/roles", limit(permAny([]string{"access:write", "settings:write"}, h.putRolesJSON)))
	mux.HandleFunc("PUT /api/v1/access/roles.yaml", limit(permAny([]string{"access:write", "settings:write"}, h.putRolesYAML)))
	mux.HandleFunc("POST /api/v1/access/roles/validate", limit(perm("access:read", h.postValidate)))
	mux.HandleFunc("POST /api/v1/access/roles/reload", limit(permAny([]string{"access:write", "settings:write"}, h.postReload)))
}

func (h *HTTPHandlers) getCatalog(w http.ResponseWriter, r *http.Request) {
	httpresponse.JSON(w, http.StatusOK, Catalog())
}

func (h *HTTPHandlers) getRoles(w http.ResponseWriter, r *http.Request) {
	doc := h.Store.Current()
	compiled := CompileDocument(doc)
	views := viewsFromDoc(doc, compiled, nil)
	scopeFilter := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("scope")))
	if scopeFilter != "" {
		filtered := make(map[string]RoleView, len(views))
		for code, view := range views {
			if view.Scope == scopeFilter {
				filtered[code] = view
			}
		}
		views = filtered
	}
	httpresponse.JSON(w, http.StatusOK, map[string]any{
		"version":  doc.Version,
		"revision": doc.Revision,
		"roles":    views,
	})
}

func (h *HTTPHandlers) getRolesYAML(w http.ResponseWriter, r *http.Request) {
	data, err := h.Store.RawYAML()
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/yaml")
	w.Header().Set("ETag", revisionETag(h.Store.Revision()))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func (h *HTTPHandlers) putRolesYAML(w http.ResponseWriter, r *http.Request) {
	body, err := coldpath.ReadLimitedBody(w, r, MaxFileBytes)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "failed to read body")
		return
	}
	expected := parseIfMatchRevision(r.Header.Get("If-Match"))
	result, err := h.Store.ApplyBytes(r.Context(), body, h.applyOpts(r, expected, false))
	if err != nil {
		h.writeApplyError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, result)
}

func (h *HTTPHandlers) putRolesJSON(w http.ResponseWriter, r *http.Request) {
	body, err := coldpath.ReadLimitedBody(w, r, MaxFileBytes)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "failed to read body")
		return
	}
	doc, err := coldpath.DecodeBody[RolesDocument](body)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid body")
		return
	}
	expected := doc.Revision
	if hdr := parseIfMatchRevision(r.Header.Get("If-Match")); hdr > 0 {
		expected = hdr
	}
	allowWildcard := strings.EqualFold(r.URL.Query().Get("allow_wildcard"), "true")
	result, err := h.Store.ApplyDocument(r.Context(), doc, h.applyOpts(r, expected, allowWildcard))
	if err != nil {
		h.writeApplyError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, result)
}

func (h *HTTPHandlers) postValidate(w http.ResponseWriter, r *http.Request) {
	body, err := coldpath.ReadLimitedBody(w, r, MaxFileBytes)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "failed to read body")
		return
	}
	allowWildcard := strings.EqualFold(r.URL.Query().Get("allow_wildcard"), "true")
	resp, err := h.Store.ValidateBytes(body, allowWildcard)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	status := http.StatusOK
	if !resp.Valid {
		status = http.StatusBadRequest
	}
	httpresponse.JSON(w, status, resp)
}

func (h *HTTPHandlers) postReload(w http.ResponseWriter, r *http.Request) {
	if err := h.Store.Reload(); err != nil {
		h.writeServiceError(w, err)
		return
	}
	doc := h.Store.Current()
	httpresponse.JSON(w, http.StatusOK, map[string]any{
		"status":   "reloaded",
		"path":     h.Store.Path(),
		"revision": doc.Revision,
	})
}

func (h *HTTPHandlers) applyOpts(r *http.Request, expectedRevision int, allowWildcard bool) ApplyOptions {
	actor := ""
	if h.ActorUserID != nil {
		if id, ok := h.ActorUserID(r); ok {
			actor = id.String()
		}
	}
	return ApplyOptions{
		ExpectedRevision: expectedRevision,
		ActorUserID:      actor,
		AllowWildcard:    allowWildcard,
		Audit: func(action string, changes, metadata any) {
			if h.Audit == nil {
				return
			}
			if id, ok := h.ActorUserID(r); ok {
				h.Audit(r.Context(), id, changes, metadata)
			}
		},
	}
}

func (h *HTTPHandlers) writeApplyError(w http.ResponseWriter, err error) {
	if details, ok := IsValidationError(err); ok {
		httpresponse.JSON(w, http.StatusBadRequest, map[string]any{
			"error":   map[string]string{"code": "BAD_REQUEST", "message": "validation failed"},
			"details": details,
		})
		return
	}
	if IsRevisionConflict(err) {
		httpresponse.Error(w, http.StatusConflict, "CONFLICT", "revision mismatch")
		return
	}
	if err != nil && strings.Contains(err.Error(), "role_in_use") {
		httpresponse.Error(w, http.StatusConflict, "role_in_use", err.Error())
		return
	}
	if err != nil && strings.Contains(err.Error(), "cannot delete builtin role") {
		httpresponse.Error(w, http.StatusForbidden, "FORBIDDEN", err.Error())
		return
	}
	h.writeServiceError(w, err)
}

func (h *HTTPHandlers) writeServiceError(w http.ResponseWriter, err error) {
	if h.WriteServiceError != nil {
		h.WriteServiceError(w, err)
		return
	}
	if errors.Is(err, io.EOF) {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "empty body")
		return
	}
	httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL", "internal error")
}

func parseIfMatchRevision(raw string) int {
	raw = strings.Trim(strings.TrimSpace(raw), "\"")
	if raw == "" {
		return 0
	}
	if strings.HasPrefix(raw, "W/") {
		raw = strings.TrimPrefix(raw, "W/")
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return 0
	}
	return n
}

func revisionETag(revision int) string {
	return strconv.Itoa(revision)
}
