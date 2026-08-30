package commandpalette

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"ad-event-processor/internal/controlplane/authz"
	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
)

type SearchAdmin interface {
	Search(ctx context.Context, customerID uuid.UUID, query string, limit int, kinds []string, licenseAllowed func(string) bool) SearchResponse
}

type HTTPHandlers struct {
	Search                         SearchAdmin
	Recents                        *RecentsStore
	LicenseFeatureAllowed          func(featureKey string) (allowed bool, planCode string)
	ApplyCommandPaletteSearchLimit func(http.HandlerFunc) http.HandlerFunc
	ApplyRateLimit                 func(http.HandlerFunc) http.HandlerFunc
	RequireAnyPermission           func([]string, http.HandlerFunc) http.HandlerFunc
	ResolveCustomerID              func(*http.Request, *uuid.UUID) (uuid.UUID, error)
	MaxQueryLen                    int
	AuditLogEnabled                bool
}

type openRequest struct {
	Source string `json:"source"`
}

func (h *HTTPHandlers) Register(mux *http.ServeMux) {
	if h == nil {
		return
	}
	searchLimit := h.ApplyCommandPaletteSearchLimit
	ipLimit := h.ApplyRateLimit
	permAny := h.RequireAnyPermission
	if searchLimit == nil {
		searchLimit = func(next http.HandlerFunc) http.HandlerFunc { return next }
	}
	if ipLimit == nil {
		ipLimit = func(next http.HandlerFunc) http.HandlerFunc { return next }
	}
	if permAny == nil {
		permAny = func(_ []string, next http.HandlerFunc) http.HandlerFunc { return next }
	}
	readPerms := []string{"campaigns:read", "campaigns:read:masked"}
	if h.Search != nil {
		mux.HandleFunc("GET /api/v1/command-palette/search", searchLimit(permAny(readPerms, h.search)))
		mux.HandleFunc("GET /api/v1/command-palette/routes", ipLimit(permAny(readPerms, h.listRoutes)))
		mux.HandleFunc("POST /api/v1/command-palette/open", ipLimit(permAny(readPerms, h.open)))
	}
	if h.Recents != nil {
		mux.HandleFunc("GET /api/v1/command-palette/recents", ipLimit(permAny(readPerms, h.listRecents)))
		mux.HandleFunc("POST /api/v1/command-palette/recents", ipLimit(permAny(readPerms, h.recordRecent)))
	}
}

func (h *HTTPHandlers) search(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	outcome := "ok"
	defer func() {
		SearchTotal.WithLabelValues(outcome).Inc()
		SearchDurationSeconds.Observe(time.Since(start).Seconds())
	}()

	customerID, err := h.resolveCustomerID(r)
	if err != nil {
		outcome = "bad_request"
		IncSearchError("bad_request")
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	q := strings.TrimSpace(r.URL.Query().Get("q"))
	maxLen := h.maxQueryLen()
	if q != "" && len(q) > maxLen {
		outcome = "bad_request"
		IncSearchError("bad_request")
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "q exceeds maximum length")
		return
	}
	if q != "" && len(q) < MinSearchQueryLen {
		outcome = "bad_request"
		IncSearchError("bad_request")
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "q must be at least 2 characters")
		return
	}

	limit := parseSearchLimit(r.URL.Query().Get("limit"))
	kinds := ParseSearchKinds(r.URL.Query()["kinds"])

	resp := h.Search.Search(r.Context(), customerID, q, limit, kinds, h.licenseAllowed)
	if resp.Items == nil {
		resp.Items = []ItemDTO{}
	}
	if resp.Degraded {
		IncSearchError("degraded")
	}
	userID := uuid.Nil
	if u, ok := authz.GetUser(r.Context()); ok {
		userID = u.UserID
	}
	logSearchAudit(h.AuditLogEnabled, customerID, userID, len(q), len(resp.Items), len(kinds), resp.Degraded)
	httpresponse.JSON(w, http.StatusOK, resp)
}

func (h *HTTPHandlers) open(w http.ResponseWriter, r *http.Request) {
	body, err := coldpath.ReadLimitedBody(w, r, coldpath.DefaultMaxBody)
	if err != nil {
		IncSearchError("bad_request")
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid body")
		return
	}
	var req openRequest
	if len(body) > 0 {
		if err := json.Unmarshal(body, &req); err != nil {
			IncSearchError("bad_request")
			httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid json")
			return
		}
	}
	source := strings.TrimSpace(req.Source)
	if source == "" {
		source = "unknown"
	}
	OpenTotal.WithLabelValues(source).Inc()
	httpresponse.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *HTTPHandlers) listRecents(w http.ResponseWriter, r *http.Request) {
	if h.Recents == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "command palette recents unavailable")
		return
	}
	u, ok := authz.GetUser(r.Context())
	if !ok || u.UserID == uuid.Nil {
		httpresponse.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized")
		return
	}
	customerID, err := h.resolveCustomerID(r)
	if err != nil {
		if isForbiddenCustomerError(err) {
			httpresponse.Error(w, http.StatusForbidden, "FORBIDDEN", err.Error())
			return
		}
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	items, err := h.Recents.ListRecents(r.Context(), customerID, u.UserID)
	if err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL", err.Error())
		return
	}
	if items == nil {
		items = []ItemDTO{}
	}
	httpresponse.JSON(w, http.StatusOK, RecentsResponse{
		Items: items,
		Total: len(items),
	})
}

func (h *HTTPHandlers) recordRecent(w http.ResponseWriter, r *http.Request) {
	if h.Recents == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "command palette recents unavailable")
		return
	}
	u, ok := authz.GetUser(r.Context())
	if !ok || u.UserID == uuid.Nil {
		httpresponse.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized")
		return
	}
	req, ok := coldpath.DecodeRequestOrBadRequest[RecordRecentRequest](w, r, coldpath.DefaultMaxBody)
	if !ok {
		return
	}
	customerID := req.CustomerID
	var err error
	if customerID == uuid.Nil {
		customerID, err = h.resolveCustomerID(r)
		if err != nil {
			if isForbiddenCustomerError(err) {
				httpresponse.Error(w, http.StatusForbidden, "FORBIDDEN", err.Error())
				return
			}
			httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
			return
		}
	} else if h.ResolveCustomerID != nil {
		customerID, err = h.ResolveCustomerID(r, &customerID)
		if err != nil {
			httpresponse.Error(w, http.StatusForbidden, "FORBIDDEN", err.Error())
			return
		}
	}
	if err := h.Recents.RecordRecent(r.Context(), customerID, u.UserID, req.Item); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *HTTPHandlers) listRoutes(w http.ResponseWriter, r *http.Request) {
	items := FilterNavCatalog(r.Context(), NavCatalogEntries(), h.licenseAllowed)
	if items == nil {
		items = []ItemDTO{}
	}
	httpresponse.JSON(w, http.StatusOK, RoutesResponse{
		Items: items,
		Total: len(items),
	})
}

func (h *HTTPHandlers) resolveCustomerID(r *http.Request) (uuid.UUID, error) {
	q := r.URL.Query()
	var customerID uuid.UUID
	if custStr := strings.TrimSpace(q.Get("customer_id")); custStr != "" {
		id, err := uuid.Parse(custStr)
		if err != nil {
			return uuid.Nil, errInvalidCustomerID
		}
		customerID = id
	}
	if h.ResolveCustomerID == nil {
		if customerID == uuid.Nil {
			return uuid.Nil, errCustomerIDRequired
		}
		return customerID, nil
	}
	resolved, err := h.ResolveCustomerID(r, customerIDPtr(customerID))
	if err != nil {
		return uuid.Nil, err
	}
	return resolved, nil
}

func (h *HTTPHandlers) licenseAllowed(featureKey string) bool {
	if h.LicenseFeatureAllowed == nil {
		return true
	}
	allowed, _ := h.LicenseFeatureAllowed(featureKey)
	return allowed
}

func (h *HTTPHandlers) maxQueryLen() int {
	if h == nil || h.MaxQueryLen <= 0 {
		return MaxSearchQueryLen
	}
	return h.MaxQueryLen
}

func parseSearchLimit(raw string) int {
	if raw == "" {
		return DefaultSearchLimit
	}
	var n int
	for i := 0; i < len(raw); i++ {
		c := raw[i]
		if c < '0' || c > '9' {
			return DefaultSearchLimit
		}
		n = n*10 + int(c-'0')
		if n > MaxSearchLimit {
			return MaxSearchLimit
		}
	}
	if n <= 0 {
		return DefaultSearchLimit
	}
	return n
}

func customerIDPtr(id uuid.UUID) *uuid.UUID {
	if id == uuid.Nil {
		return nil
	}
	return &id
}
