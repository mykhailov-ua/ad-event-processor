package controlplane

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"ad-event-processor/internal/dashboardadmin"
	"ad-event-processor/internal/reportjob"
	"ad-event-processor/internal/reports"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRouteRegistration(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	reportsHandler := &reports.ReportsHTTPHandlers{}
	dashboardsHandler := &dashboardadmin.HTTPHandlers{}
	viewsHandler := &reports.ViewsHTTPHandlers{Store: reports.NewViewsStore(nil)}

	registry := RouteRegistry{
		ReportsHTTP:    reportsHandler,
		ReportJobHTTP:  &reportjob.HTTPHandlers{Runner: &reportjob.ReportJobRunner{}},
		DashboardsHTTP: dashboardsHandler,
		ViewsHTTP:      viewsHandler,
	}

	RegisterRoutes(mux, registry)

	routes := []struct {
		method string
		path   string
	}{
		{"GET", "/api/v1/reports/placements"},
		{"GET", "/api/v1/reports/keywords"},
		{"GET", "/api/v1/dashboards/campaign/" + uuid.New().String()},
		{"GET", "/api/v1/views?customer_id=" + uuid.New().String()},
		{"POST", "/api/v1/views"},
	}

	for _, rt := range routes {
		req := httptest.NewRequest(rt.method, rt.path, http.NoBody)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		assert.NotEqual(t, http.StatusNotFound, w.Code, "route %s %s not registered", rt.method, rt.path)
	}
}

func TestDashboards_Campaign_requiresConfiguredReader(t *testing.T) {
	t.Parallel()

	h := &dashboardadmin.HTTPHandlers{}
	mux := http.NewServeMux()
	h.Register(mux)

	campaignID := uuid.New()
	req := httptest.NewRequest("GET", "/api/v1/dashboards/campaign/"+campaignID.String(), http.NoBody)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotImplemented, w.Code)
}

func TestViews_CRUD(t *testing.T) {
	t.Parallel()

	h := &reports.ViewsHTTPHandlers{Store: reports.NewViewsStore(nil)}
	mux := http.NewServeMux()
	h.Register(mux)

	customerID := uuid.New().String()

	createReq := reports.CreateViewRequest{
		CustomerID: customerID,
		Name:       "My Placement View",
		ReportKey:  "placements",
		Spec:       json.RawMessage(`{"limit":10}`),
		IsShared:   true,
	}
	body, _ := json.Marshal(createReq)
	req := httptest.NewRequest("POST", "/api/v1/views", bytes.NewReader(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code)

	var created reports.SavedViewDTO
	err := json.Unmarshal(w.Body.Bytes(), &created)
	require.NoError(t, err)

	assert.NotEmpty(t, created.ID)
	assert.Equal(t, createReq.Name, created.Name)
	assert.Equal(t, createReq.CustomerID, created.CustomerID)

	reqList := httptest.NewRequest("GET", "/api/v1/views?customer_id="+customerID, http.NoBody)
	wList := httptest.NewRecorder()
	mux.ServeHTTP(wList, reqList)

	require.Equal(t, http.StatusOK, wList.Code)

	var list []reports.SavedViewDTO
	err = json.Unmarshal(wList.Body.Bytes(), &list)
	require.NoError(t, err)

	assert.Len(t, list, 1)
	assert.Equal(t, created.ID, list[0].ID)

	reqGet := httptest.NewRequest("GET", "/api/v1/views/"+created.ID, http.NoBody)
	wGet := httptest.NewRecorder()
	mux.ServeHTTP(wGet, reqGet)

	require.Equal(t, http.StatusOK, wGet.Code)

	var fetched reports.SavedViewDTO
	err = json.Unmarshal(wGet.Body.Bytes(), &fetched)
	require.NoError(t, err)

	assert.Equal(t, created.ID, fetched.ID)

	updateReq := reports.UpdateViewRequest{
		Name:      "Updated View Name",
		ReportKey: "placements",
		Spec:      json.RawMessage(`{"limit":20}`),
		IsShared:  false,
	}
	updateBody, _ := json.Marshal(updateReq)
	reqUpdate := httptest.NewRequest("PUT", "/api/v1/views/"+created.ID, bytes.NewReader(updateBody))
	wUpdate := httptest.NewRecorder()
	mux.ServeHTTP(wUpdate, reqUpdate)

	require.Equal(t, http.StatusOK, wUpdate.Code)

	var updated reports.SavedViewDTO
	err = json.Unmarshal(wUpdate.Body.Bytes(), &updated)
	require.NoError(t, err)

	assert.Equal(t, updateReq.Name, updated.Name)
	assert.False(t, updated.IsShared)

	reqDelete := httptest.NewRequest("DELETE", "/api/v1/views/"+created.ID, http.NoBody)
	wDelete := httptest.NewRecorder()
	mux.ServeHTTP(wDelete, reqDelete)

	require.Equal(t, http.StatusNoContent, wDelete.Code)

	reqGet2 := httptest.NewRequest("GET", "/api/v1/views/"+created.ID, http.NoBody)
	wGet2 := httptest.NewRecorder()
	mux.ServeHTTP(wGet2, reqGet2)

	assert.Equal(t, http.StatusNotFound, wGet2.Code)
}

func TestViews_CustomerAccessDenied(t *testing.T) {
	t.Parallel()

	customerID := uuid.New().String()
	otherCustomer := uuid.New().String()

	h := &reports.ViewsHTTPHandlers{
		Store: reports.NewViewsStore(nil),
		AuthorizeCustomerAccess: func(_ *http.Request, custID string) error {
			if custID != customerID {
				return ErrForbidden
			}
			return nil
		},
		WriteServiceError: func(w http.ResponseWriter, err error) {
			if errors.Is(err, ErrForbidden) {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
		},
	}
	mux := http.NewServeMux()
	h.Register(mux)

	createReq := reports.CreateViewRequest{
		CustomerID: otherCustomer,
		Name:       "blocked",
		ReportKey:  "placements",
		Spec:       json.RawMessage(`{}`),
	}
	body, _ := json.Marshal(createReq)
	req := httptest.NewRequest("POST", "/api/v1/views", bytes.NewReader(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	require.Equal(t, http.StatusForbidden, w.Code)

	reqList := httptest.NewRequest("GET", "/api/v1/views?customer_id="+otherCustomer, http.NoBody)
	wList := httptest.NewRecorder()
	mux.ServeHTTP(wList, reqList)
	require.Equal(t, http.StatusForbidden, wList.Code)

	allowed := reports.CreateViewRequest{
		CustomerID: customerID,
		Name:       "allowed",
		ReportKey:  "placements",
		Spec:       json.RawMessage(`{}`),
	}
	allowedBody, _ := json.Marshal(allowed)
	reqOK := httptest.NewRequest("POST", "/api/v1/views", bytes.NewReader(allowedBody))
	wOK := httptest.NewRecorder()
	mux.ServeHTTP(wOK, reqOK)
	require.Equal(t, http.StatusCreated, wOK.Code)
}

func TestViews_ValidationRejectsUnknownSpecKey(t *testing.T) {
	t.Parallel()

	h := &reports.ViewsHTTPHandlers{Store: reports.NewViewsStore(nil)}
	mux := http.NewServeMux()
	h.Register(mux)

	body, _ := json.Marshal(reports.CreateViewRequest{
		CustomerID: uuid.New().String(),
		Name:       "bad spec",
		ReportKey:  "placements",
		Spec:       json.RawMessage(`{"api_token":"secret"}`),
	})
	req := httptest.NewRequest("POST", "/api/v1/views", bytes.NewReader(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}
