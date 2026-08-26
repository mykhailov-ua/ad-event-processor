package controlplane

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync"
	"time"

	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SavedViewDTO struct {
	ID         string          `json:"id"`
	OwnerID    string          `json:"owner_id"`
	CustomerID string          `json:"customer_id"`
	Name       string          `json:"name"`
	ReportKey  string          `json:"report_key"`
	Spec       json.RawMessage `json:"spec"`
	IsShared   bool            `json:"is_shared"`
	CreatedAt  string          `json:"created_at"`
	UpdatedAt  string          `json:"updated_at"`
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

func (s *ViewsStore) CreateView(ctx context.Context, req CreateViewRequest, ownerID string) (SavedViewDTO, error) {
	if s.pgEnabled() {
		return s.createViewPG(ctx, req, ownerID)
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
		ID:         id,
		OwnerID:    ownerID,
		CustomerID: req.CustomerID,
		Name:       req.Name,
		ReportKey:  req.ReportKey,
		Spec:       spec,
		IsShared:   req.IsShared,
		CreatedAt:  now,
		UpdatedAt:  now,
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

func (s *ViewsStore) UpdateView(ctx context.Context, id string, req UpdateViewRequest) (SavedViewDTO, error) {
	if s.pgEnabled() {
		return s.updateViewPG(ctx, id, req)
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

func (viewHandlers *ViewsHTTPHandlers) authorizeViewCustomer(w http.ResponseWriter, r *http.Request, customerID string) bool {
	if viewHandlers == nil || viewHandlers.AuthorizeCustomerAccess == nil {
		return true
	}
	if err := viewHandlers.AuthorizeCustomerAccess(r, customerID); err != nil {
		viewHandlers.writeServiceError(w, err)
		return false
	}
	return true
}

func (viewHandlers *ViewsHTTPHandlers) writeServiceError(w http.ResponseWriter, err error) {
	if viewHandlers != nil && viewHandlers.WriteServiceError != nil {
		viewHandlers.WriteServiceError(w, err)
		return
	}
	httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
}

func (viewHandlers *ViewsHTTPHandlers) Register(mux *http.ServeMux) {
	if viewHandlers == nil {
		return
	}
	limit := viewHandlers.ApplyRateLimit
	perm := viewHandlers.RequirePermission
	permAny := viewHandlers.RequireAnyPermission
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
	mux.HandleFunc("GET /api/v1/views", limit(permAny(readPerms, viewHandlers.listViews)))
	mux.HandleFunc("GET /api/v1/views/{id}", limit(permAny(readPerms, viewHandlers.getView)))
	mux.HandleFunc("POST /api/v1/views", limit(perm("campaigns:write", viewHandlers.createView)))
	mux.HandleFunc("PUT /api/v1/views/{id}", limit(perm("campaigns:write", viewHandlers.updateView)))
	mux.HandleFunc("DELETE /api/v1/views/{id}", limit(perm("campaigns:write", viewHandlers.deleteView)))
}

func (viewHandlers *ViewsHTTPHandlers) createView(w http.ResponseWriter, r *http.Request) {
	req, err := coldpath.DecodeRequest[CreateViewRequest](w, r, coldpath.DefaultMaxBody)
	if err != nil {
		return
	}

	if _, err := uuid.Parse(req.CustomerID); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid customer_id")
		return
	}
	if !viewHandlers.authorizeViewCustomer(w, r, req.CustomerID) {
		return
	}
	if err := validateSavedViewInput(req.Name, req.ReportKey, req.Spec); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	if viewHandlers.Store == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "UNAVAILABLE", "views service not configured")
		return
	}

	ownerID := "system"
	view, err := viewHandlers.Store.CreateView(r.Context(), req, ownerID)
	if err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	httpresponse.JSON(w, http.StatusCreated, view)
}

func (viewHandlers *ViewsHTTPHandlers) listViews(w http.ResponseWriter, r *http.Request) {
	custIDStr := r.URL.Query().Get("customer_id")
	if custIDStr == "" {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "customer_id query parameter is required")
		return
	}

	if _, err := uuid.Parse(custIDStr); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid customer_id")
		return
	}
	if !viewHandlers.authorizeViewCustomer(w, r, custIDStr) {
		return
	}

	if viewHandlers.Store == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "UNAVAILABLE", "views service not configured")
		return
	}

	views, err := viewHandlers.Store.ListView(r.Context(), custIDStr)
	if err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	httpresponse.JSON(w, http.StatusOK, views)
}

func (viewHandlers *ViewsHTTPHandlers) getView(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "missing view id")
		return
	}

	if viewHandlers.Store == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "UNAVAILABLE", "views service not configured")
		return
	}

	view, err := viewHandlers.Store.GetView(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrViewNotFound) {
			httpresponse.Error(w, http.StatusNotFound, "NOT_FOUND", "view not found")
			return
		}
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	if !viewHandlers.authorizeViewCustomer(w, r, view.CustomerID) {
		return
	}

	httpresponse.JSON(w, http.StatusOK, view)
}

func (viewHandlers *ViewsHTTPHandlers) updateView(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "missing view id")
		return
	}

	req, err := coldpath.DecodeRequest[UpdateViewRequest](w, r, coldpath.DefaultMaxBody)
	if err != nil {
		return
	}

	if viewHandlers.Store == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "UNAVAILABLE", "views service not configured")
		return
	}

	existing, err := viewHandlers.Store.GetView(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrViewNotFound) {
			httpresponse.Error(w, http.StatusNotFound, "NOT_FOUND", "view not found")
			return
		}
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	if !viewHandlers.authorizeViewCustomer(w, r, existing.CustomerID) {
		return
	}
	if err := validateSavedViewInput(req.Name, req.ReportKey, req.Spec); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	updated, err := viewHandlers.Store.UpdateView(r.Context(), id, req)
	if err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	httpresponse.JSON(w, http.StatusOK, updated)
}

func (viewHandlers *ViewsHTTPHandlers) deleteView(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "missing view id")
		return
	}

	if viewHandlers.Store == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "UNAVAILABLE", "views service not configured")
		return
	}

	existing, err := viewHandlers.Store.GetView(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrViewNotFound) {
			httpresponse.Error(w, http.StatusNotFound, "NOT_FOUND", "view not found")
			return
		}
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	if !viewHandlers.authorizeViewCustomer(w, r, existing.CustomerID) {
		return
	}

	if err := viewHandlers.Store.DeleteView(r.Context(), id); err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
