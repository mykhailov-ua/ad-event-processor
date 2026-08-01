package adminapi

import (
	"errors"
	"net/http"
	"sync"
	"time"

	"espx/internal/ledger/db"
	"espx/pkg/coldpath"
	"espx/pkg/httpresponse"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SavedViewDTO struct {
	ID         string         `json:"id"`
	OwnerID    string         `json:"owner_id"`
	CustomerID string         `json:"customer_id"`
	Name       string         `json:"name"`
	ReportKey  string         `json:"report_key"`
	Spec       map[string]any `json:"spec"`
	IsShared   bool           `json:"is_shared"`
	CreatedAt  string         `json:"created_at"`
	UpdatedAt  string         `json:"updated_at"`
}

type CreateViewRequest struct {
	CustomerID string         `json:"customer_id"`
	Name       string         `json:"name"`
	ReportKey  string         `json:"report_key"`
	Spec       map[string]any `json:"spec"`
	IsShared   bool           `json:"is_shared"`
}

type UpdateViewRequest struct {
	Name      string         `json:"name"`
	ReportKey string         `json:"report_key"`
	Spec      map[string]any `json:"spec"`
	IsShared  bool           `json:"is_shared"`
}

var (
	ErrViewNotFound = errors.New("view not found")
)

type Service struct {
	mu    sync.RWMutex
	views map[string]SavedViewDTO
}

func NewService() *Service {
	return &Service{
		views: make(map[string]SavedViewDTO),
	}
}

func (s *Service) CreateView(req CreateViewRequest, ownerID string) SavedViewDTO {
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

func (s *Service) GetView(id string) (SavedViewDTO, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	view, ok := s.views[id]
	if !ok {
		return SavedViewDTO{}, ErrViewNotFound
	}
	return view, nil
}

func (s *Service) ListView(customerID string) []SavedViewDTO {
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

func (s *Service) UpdateView(id string, req UpdateViewRequest) (SavedViewDTO, error) {
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

func (s *Service) DeleteView(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.views[id]; !ok {
		return ErrViewNotFound
	}
	delete(s.views, id)
	return nil
}

type ViewsHTTPHandlers struct {
	Service           *Service
	Pool              *pgxpool.Pool
	ApplyRateLimit    func(http.HandlerFunc) http.HandlerFunc
	RequirePermission func(string, http.HandlerFunc) http.HandlerFunc
}

func (viewHandlers *ViewsHTTPHandlers) Register(mux *http.ServeMux) {
	if viewHandlers == nil {
		return
	}
	limit := viewHandlers.ApplyRateLimit
	perm := viewHandlers.RequirePermission
	if limit == nil {
		limit = func(next http.HandlerFunc) http.HandlerFunc { return next }
	}
	if perm == nil {
		perm = func(_ string, next http.HandlerFunc) http.HandlerFunc { return next }
	}

	mux.HandleFunc("GET /api/v1/views", limit(perm("campaigns:read", viewHandlers.listViews)))
	mux.HandleFunc("POST /api/v1/views", limit(perm("campaigns:write", viewHandlers.createView)))
	mux.HandleFunc("GET /api/v1/views/{id}", limit(perm("campaigns:read", viewHandlers.getView)))
	mux.HandleFunc("PUT /api/v1/views/{id}", limit(perm("campaigns:write", viewHandlers.updateView)))
	mux.HandleFunc("DELETE /api/v1/views/{id}", limit(perm("campaigns:write", viewHandlers.deleteView)))
}

func (viewHandlers *ViewsHTTPHandlers) checkTierGate(r *http.Request, customerID uuid.UUID) (bool, error) {
	if viewHandlers.Pool == nil {
		return true, nil
	}
	q := db.New(viewHandlers.Pool)
	sub, err := q.GetCustomerSubscription(r.Context(), pgtype.UUID{Bytes: customerID, Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return sub.PlanCode == "enterprise", nil
}

func (viewHandlers *ViewsHTTPHandlers) createView(w http.ResponseWriter, r *http.Request) {
	req, err := coldpath.DecodeRequest[CreateViewRequest](w, r, coldpath.DefaultMaxBody)
	if err != nil {
		return
	}

	customerID, err := uuid.Parse(req.CustomerID)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid customer_id")
		return
	}

	allowed, err := viewHandlers.checkTierGate(r, customerID)
	if err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	if !allowed {
		httpresponse.Error(w, http.StatusForbidden, "FORBIDDEN", "Enterprise plan required")
		return
	}

	if viewHandlers.Service == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "UNAVAILABLE", "views service not configured")
		return
	}

	ownerID := "system"
	view := viewHandlers.Service.CreateView(req, ownerID)
	httpresponse.JSON(w, http.StatusCreated, view)
}

func (viewHandlers *ViewsHTTPHandlers) listViews(w http.ResponseWriter, r *http.Request) {
	custIDStr := r.URL.Query().Get("customer_id")
	if custIDStr == "" {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "customer_id query parameter is required")
		return
	}

	customerID, err := uuid.Parse(custIDStr)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid customer_id")
		return
	}

	allowed, err := viewHandlers.checkTierGate(r, customerID)
	if err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	if !allowed {
		httpresponse.Error(w, http.StatusForbidden, "FORBIDDEN", "Enterprise plan required")
		return
	}

	if viewHandlers.Service == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "UNAVAILABLE", "views service not configured")
		return
	}

	views := viewHandlers.Service.ListView(custIDStr)
	httpresponse.JSON(w, http.StatusOK, views)
}

func (viewHandlers *ViewsHTTPHandlers) getView(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "missing view id")
		return
	}

	if viewHandlers.Service == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "UNAVAILABLE", "views service not configured")
		return
	}

	view, err := viewHandlers.Service.GetView(id)
	if err != nil {
		if errors.Is(err, ErrViewNotFound) {
			httpresponse.Error(w, http.StatusNotFound, "NOT_FOUND", "view not found")
			return
		}
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	customerID, _ := uuid.Parse(view.CustomerID)
	allowed, err := viewHandlers.checkTierGate(r, customerID)
	if err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	if !allowed {
		httpresponse.Error(w, http.StatusForbidden, "FORBIDDEN", "Enterprise plan required")
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

	if viewHandlers.Service == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "UNAVAILABLE", "views service not configured")
		return
	}

	existing, err := viewHandlers.Service.GetView(id)
	if err != nil {
		if errors.Is(err, ErrViewNotFound) {
			httpresponse.Error(w, http.StatusNotFound, "NOT_FOUND", "view not found")
			return
		}
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	customerID, _ := uuid.Parse(existing.CustomerID)
	allowed, err := viewHandlers.checkTierGate(r, customerID)
	if err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	if !allowed {
		httpresponse.Error(w, http.StatusForbidden, "FORBIDDEN", "Enterprise plan required")
		return
	}

	updated, err := viewHandlers.Service.UpdateView(id, req)
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

	if viewHandlers.Service == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "UNAVAILABLE", "views service not configured")
		return
	}

	existing, err := viewHandlers.Service.GetView(id)
	if err != nil {
		if errors.Is(err, ErrViewNotFound) {
			httpresponse.Error(w, http.StatusNotFound, "NOT_FOUND", "view not found")
			return
		}
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	customerID, _ := uuid.Parse(existing.CustomerID)
	allowed, err := viewHandlers.checkTierGate(r, customerID)
	if err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	if !allowed {
		httpresponse.Error(w, http.StatusForbidden, "FORBIDDEN", "Enterprise plan required")
		return
	}

	err = viewHandlers.Service.DeleteView(id)
	if err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
