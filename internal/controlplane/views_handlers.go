package controlplane

import (
	"encoding/json"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/bidshard/ad-event-processor/pkg/coldpath"
	"github.com/bidshard/ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
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
	mu    sync.RWMutex
	views map[string]SavedViewDTO
}

func NewViewsStore() *ViewsStore {
	return &ViewsStore{
		views: make(map[string]SavedViewDTO),
	}
}

func (s *ViewsStore) CreateView(req CreateViewRequest, ownerID string) SavedViewDTO {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := uuid.New().String()
	now := time.Now().UTC().Format(time.RFC3339)
	view := SavedViewDTO{
		ID:         id,
		OwnerID:    ownerID,
		CustomerID: req.CustomerID,
		Name:       req.Name,
		ReportKey:  req.ReportKey,
		Spec:       req.Spec,
		IsShared:   req.IsShared,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	s.views[id] = view
	return view
}

func (s *ViewsStore) GetView(id string) (SavedViewDTO, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	view, ok := s.views[id]
	if !ok {
		return SavedViewDTO{}, ErrViewNotFound
	}
	return view, nil
}

func (s *ViewsStore) ListView(customerID string) []SavedViewDTO {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var list []SavedViewDTO
	for _, v := range s.views {
		if v.CustomerID == customerID {
			list = append(list, v)
		}
	}
	return list
}

func (s *ViewsStore) UpdateView(id string, req UpdateViewRequest) (SavedViewDTO, error) {
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

func (s *ViewsStore) DeleteView(id string) error {
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

	if viewHandlers.Store == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "UNAVAILABLE", "views service not configured")
		return
	}

	ownerID := "system"
	view := viewHandlers.Store.CreateView(req, ownerID)
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

	views := viewHandlers.Store.ListView(custIDStr)
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

	view, err := viewHandlers.Store.GetView(id)
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

	existing, err := viewHandlers.Store.GetView(id)
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

	updated, err := viewHandlers.Store.UpdateView(id, req)
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

	existing, err := viewHandlers.Store.GetView(id)
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

	if err := viewHandlers.Store.DeleteView(id); err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
