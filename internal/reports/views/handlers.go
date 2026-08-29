package views

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync"
	"time"

	"ad-event-processor/internal/controlplane/authz"
	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SavedViewDTO struct {
	ID             string          `json:"id"`
	OwnerID        string          `json:"owner_id"`
	OwnerMaskLevel string          `json:"owner_mask_level,omitempty"`
	CustomerID     string          `json:"customer_id"`
	Name           string          `json:"name"`
	ReportKey      string          `json:"report_key"`
	Spec           json.RawMessage `json:"spec"`
	IsShared       bool            `json:"is_shared"`
	CreatedAt      string          `json:"created_at"`
	UpdatedAt      string          `json:"updated_at"`
}

type CreateViewRequest struct {
	CustomerID string          `json:"customer_id"`
	Name       string          `json:"name"`
	ReportKey  string          `json:"report_key"`
	Spec       json.RawMessage `json:"spec"`
	IsShared   bool            `json:"is_shared"`
}

type UpdateViewRequest struct {
	Name      string          `json:"name"`
	ReportKey string          `json:"report_key"`
	Spec      json.RawMessage `json:"spec"`
	IsShared  bool            `json:"is_shared"`
}

var ErrViewNotFound = errors.New("view not found")

type ViewsStore struct {
	pool  *pgxpool.Pool
	mu    sync.RWMutex
	views map[string]SavedViewDTO
}

func NewViewsStore(pool *pgxpool.Pool) *ViewsStore {
	return &ViewsStore{
		pool:  pool,
		views: make(map[string]SavedViewDTO),
	}
}

func (s *ViewsStore) CreateView(ctx context.Context, req CreateViewRequest, ownerID string, ownerMask authz.MaskLevel) (SavedViewDTO, error) {
	if s.pgEnabled() {
		return s.createViewPG(ctx, req, ownerID, ownerMask)
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	id := uuid.New().String()
	now := time.Now().UTC().Format(time.RFC3339)
	spec := req.Spec
	if len(spec) == 0 {
		spec = json.RawMessage(`{}`)
	}
	view := SavedViewDTO{
		ID:             id,
		OwnerID:        ownerID,
		OwnerMaskLevel: string(ownerMask),
		CustomerID:     req.CustomerID,
		Name:           req.Name,
		ReportKey:      req.ReportKey,
		Spec:           spec,
		IsShared:       req.IsShared,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	s.views[id] = view
	return view, nil
}

func (s *ViewsStore) GetView(ctx context.Context, id string) (SavedViewDTO, error) {
	if s.pgEnabled() {
		return s.getViewPG(ctx, id)
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	view, ok := s.views[id]
	if !ok {
		return SavedViewDTO{}, ErrViewNotFound
	}
	return view, nil
}

func (s *ViewsStore) ListView(ctx context.Context, customerID string) ([]SavedViewDTO, error) {
	if s.pgEnabled() {
		return s.listViewsPG(ctx, customerID)
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	list := make([]SavedViewDTO, 0, len(s.views))
	for _, v := range s.views {
		if v.CustomerID == customerID {
			list = append(list, v)
		}
	}
	return list, nil
}

func (s *ViewsStore) UpdateView(ctx context.Context, id string, req UpdateViewRequest, ownerMask authz.MaskLevel) (SavedViewDTO, error) {
	if s.pgEnabled() {
		return s.updateViewPG(ctx, id, req, ownerMask)
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	view, ok := s.views[id]
	if !ok {
		return SavedViewDTO{}, ErrViewNotFound
	}

	view.Name = req.Name
	view.ReportKey = req.ReportKey
	view.Spec = req.Spec
	view.IsShared = req.IsShared
	view.OwnerMaskLevel = string(ownerMask)
	view.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	s.views[id] = view
	return view, nil
}

func (s *ViewsStore) DeleteView(ctx context.Context, id string) error {
	if s.pgEnabled() {
		return s.deleteViewPG(ctx, id)
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.views[id]; !ok {
		return ErrViewNotFound
	}
	delete(s.views, id)
	return nil
}

type ViewsHTTPHandlers struct {
	Store                   *ViewsStore
	ApplyRateLimit          func(http.HandlerFunc) http.HandlerFunc
	RequirePermission       func(string, http.HandlerFunc) http.HandlerFunc
	RequireAnyPermission    func([]string, http.HandlerFunc) http.HandlerFunc
	AuthorizeCustomerAccess func(*http.Request, string) error
	WriteServiceError       func(http.ResponseWriter, error)
}

func (h *ViewsHTTPHandlers) authorizeViewCustomer(w http.ResponseWriter, r *http.Request, customerID string) bool {
	if h == nil || h.AuthorizeCustomerAccess == nil {
		return true
	}
	if err := h.AuthorizeCustomerAccess(r, customerID); err != nil {
		h.writeServiceError(w, err)
		return false
	}
	return true
}

func (h *ViewsHTTPHandlers) writeServiceError(w http.ResponseWriter, err error) {
	if h != nil && h.WriteServiceError != nil {
		h.WriteServiceError(w, err)
		return
	}
	httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
}

func (h *ViewsHTTPHandlers) Register(mux *http.ServeMux) {
	if h == nil {
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
			return perm(perms[0], next)
		}
	}

	readPerms := []string{"campaigns:read", "campaigns:read:masked"}
	mux.HandleFunc("GET /api/v1/views", limit(permAny(readPerms, h.listViews)))
	mux.HandleFunc("GET /api/v1/views/{id}", limit(permAny(readPerms, h.getView)))
	mux.HandleFunc("POST /api/v1/views", limit(perm("campaigns:write", h.createView)))
	mux.HandleFunc("PUT /api/v1/views/{id}", limit(perm("campaigns:write", h.updateView)))
	mux.HandleFunc("DELETE /api/v1/views/{id}", limit(perm("campaigns:write", h.deleteView)))
}

func (h *ViewsHTTPHandlers) createView(w http.ResponseWriter, r *http.Request) {
	req, err := coldpath.DecodeRequest[CreateViewRequest](w, r, coldpath.DefaultMaxBody)
	if err != nil {
		return
	}

	if _, err := uuid.Parse(req.CustomerID); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid customer_id")
		return
	}
	if !h.authorizeViewCustomer(w, r, req.CustomerID) {
		return
	}
	if err := validateSavedViewInputForActor(r.Context(), req.Name, req.ReportKey, req.Spec); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	if h.Store == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "UNAVAILABLE", "views service not configured")
		return
	}

	ownerID := "system"
	if u, ok := authz.GetUser(r.Context()); ok && u.UserID != uuid.Nil {
		ownerID = u.UserID.String()
	}
	ownerMask := savedViewMaskFromContext(r.Context())
	view, err := h.Store.CreateView(r.Context(), req, ownerID, ownerMask)
	if err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	httpresponse.JSON(w, http.StatusCreated, view)
}

func (h *ViewsHTTPHandlers) listViews(w http.ResponseWriter, r *http.Request) {
	custIDStr := r.URL.Query().Get("customer_id")
	if custIDStr == "" {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "customer_id query parameter is required")
		return
	}

	if _, err := uuid.Parse(custIDStr); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid customer_id")
		return
	}
	if !h.authorizeViewCustomer(w, r, custIDStr) {
		return
	}

	if h.Store == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "UNAVAILABLE", "views service not configured")
		return
	}

	views, err := h.Store.ListView(r.Context(), custIDStr)
	if err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	filtered := make([]SavedViewDTO, 0, len(views))
	for i := range views {
		if err := validateSharedSavedViewForActor(r.Context(), views[i]); err != nil {
			continue
		}
		filtered = append(filtered, views[i])
	}
	httpresponse.JSON(w, http.StatusOK, filtered)
}

func (h *ViewsHTTPHandlers) getView(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "missing view id")
		return
	}

	if h.Store == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "UNAVAILABLE", "views service not configured")
		return
	}

	view, err := h.Store.GetView(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrViewNotFound) {
			httpresponse.Error(w, http.StatusNotFound, "NOT_FOUND", "view not found")
			return
		}
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	if !h.authorizeViewCustomer(w, r, view.CustomerID) {
		return
	}
	if err := validateSharedSavedViewForActor(r.Context(), view); err != nil {
		httpresponse.Error(w, http.StatusForbidden, "FORBIDDEN", err.Error())
		return
	}

	httpresponse.JSON(w, http.StatusOK, view)
}

func (h *ViewsHTTPHandlers) updateView(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "missing view id")
		return
	}

	req, err := coldpath.DecodeRequest[UpdateViewRequest](w, r, coldpath.DefaultMaxBody)
	if err != nil {
		return
	}

	if h.Store == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "UNAVAILABLE", "views service not configured")
		return
	}

	existing, err := h.Store.GetView(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrViewNotFound) {
			httpresponse.Error(w, http.StatusNotFound, "NOT_FOUND", "view not found")
			return
		}
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	if !h.authorizeViewCustomer(w, r, existing.CustomerID) {
		return
	}
	if err := validateSavedViewInputForActor(r.Context(), req.Name, req.ReportKey, req.Spec); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	ownerMask := authz.MaskLevel(existing.OwnerMaskLevel)
	if ownerMask == "" {
		ownerMask = authz.MaskMasked
	}
	if req.IsShared {
		ownerMask = savedViewMaskFromContext(r.Context())
	}

	updated, err := h.Store.UpdateView(r.Context(), id, req, ownerMask)
	if err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	httpresponse.JSON(w, http.StatusOK, updated)
}

func (h *ViewsHTTPHandlers) deleteView(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "missing view id")
		return
	}

	if h.Store == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "UNAVAILABLE", "views service not configured")
		return
	}

	existing, err := h.Store.GetView(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrViewNotFound) {
			httpresponse.Error(w, http.StatusNotFound, "NOT_FOUND", "view not found")
			return
		}
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	if !h.authorizeViewCustomer(w, r, existing.CustomerID) {
		return
	}

	if err := h.Store.DeleteView(r.Context(), id); err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
