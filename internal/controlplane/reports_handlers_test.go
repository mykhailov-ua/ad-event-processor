package controlplane

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRouteRegistration(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	reportsHandler := &ReportsHTTPHandlers{}
	dashboardsHandler := &DashboardsHTTPHandlers{}
	viewsHandler := &ViewsHTTPHandlers{Store: NewViewsStore(nil)}

	registry := RouteRegistry{
		ReportsHTTP:    reportsHandler,
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

func TestReports_Placements(t *testing.T) {
	t.Parallel()

	h := &ReportsHTTPHandlers{}
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest("GET", "/api/v1/reports/placements?customer_id="+uuid.New().String()+"&limit=5", http.NoBody)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	require.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestReports_Keywords(t *testing.T) {
	t.Parallel()

	h := &ReportsHTTPHandlers{}
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest("GET", "/api/v1/reports/keywords?customer_id="+uuid.New().String()+"&limit=5", http.NoBody)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	require.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestReports_JobRoutesRegistered(t *testing.T) {
	t.Parallel()

	h := &ReportsHTTPHandlers{ReportJobs: &ReportJobRunner{}}
	mux := http.NewServeMux()
	h.Register(mux)

	jobID := uuid.New().String()
	for _, tc := range []struct {
		method string
		path   string
	}{
		{"POST", "/api/v1/reports/jobs"},
		{"GET", "/api/v1/reports/jobs/" + jobID},
		{"GET", "/api/v1/reports/jobs/" + jobID + "/download"},
	} {
		req := httptest.NewRequest(tc.method, tc.path, http.NoBody)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		require.True(t, routeRegistered(w), "%s %s", tc.method, tc.path)
	}
}

func TestDashboards_Campaign(t *testing.T) {
	t.Parallel()

	h := &DashboardsHTTPHandlers{}
	mux := http.NewServeMux()
	h.Register(mux)

	campaignID := uuid.New()
	req := httptest.NewRequest("GET", "/api/v1/dashboards/campaign/"+campaignID.String(), http.NoBody)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp CampaignDashboardDTO
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.Equal(t, campaignID.String(), resp.CampaignID)
	assert.Equal(t, int64(150000000), resp.KPIs.SpendMicro)
	assert.Equal(t, int64(180000000), resp.KPIs.RevenueMicro)
	assert.True(t, resp.Freshness.Stale)
}

func TestViews_CRUD(t *testing.T) {
	t.Parallel()

	h := &ViewsHTTPHandlers{Store: NewViewsStore(nil)}
	mux := http.NewServeMux()
	h.Register(mux)

	customerID := uuid.New().String()

	createReq := CreateViewRequest{
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

	var created SavedViewDTO
	err := json.Unmarshal(w.Body.Bytes(), &created)
	require.NoError(t, err)

	assert.NotEmpty(t, created.ID)
	assert.Equal(t, createReq.Name, created.Name)
	assert.Equal(t, createReq.CustomerID, created.CustomerID)

	reqList := httptest.NewRequest("GET", "/api/v1/views?customer_id="+customerID, http.NoBody)
	wList := httptest.NewRecorder()
	mux.ServeHTTP(wList, reqList)

	require.Equal(t, http.StatusOK, wList.Code)

	var list []SavedViewDTO
	err = json.Unmarshal(wList.Body.Bytes(), &list)
	require.NoError(t, err)

	assert.Len(t, list, 1)
	assert.Equal(t, created.ID, list[0].ID)

	reqGet := httptest.NewRequest("GET", "/api/v1/views/"+created.ID, http.NoBody)
	wGet := httptest.NewRecorder()
	mux.ServeHTTP(wGet, reqGet)

	require.Equal(t, http.StatusOK, wGet.Code)

	var fetched SavedViewDTO
	err = json.Unmarshal(wGet.Body.Bytes(), &fetched)
	require.NoError(t, err)

	assert.Equal(t, created.ID, fetched.ID)

	updateReq := UpdateViewRequest{
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

	var updated SavedViewDTO
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

	h := &ViewsHTTPHandlers{
		Store: NewViewsStore(nil),
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

	createReq := CreateViewRequest{
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

	allowed := CreateViewRequest{
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

func TestToPlacementReportRowDTO(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		row  reportMetricsCHRow
		want PlacementReportRowDTO
	}{
		{
			name: "profit and roi",
			row: reportMetricsCHRow{
				Dimension:    "zone_1001",
				CampaignID:   "camp-1",
				Impressions:  10000,
				Clicks:       500,
				Conversions:  10,
				SpendMicro:   50_000_000,
				RevenueMicro: 60_000_000,
			},
			want: PlacementReportRowDTO{
				PlacementID:  "zone_1001",
				CampaignID:   "camp-1",
				Impressions:  10000,
				Clicks:       500,
				Conversions:  10,
				SpendMicro:   50_000_000,
				RevenueMicro: 60_000_000,
				ProfitMicro:  10_000_000,
				ROIPct:       20,
				CPAMicro:     5_000_000,
				CTR:          0.05,
			},
		},
		{
			name: "zero spend skips roi",
			row: reportMetricsCHRow{
				Dimension:  "zone_0",
				CampaignID: "camp-2",
			},
			want: PlacementReportRowDTO{
				PlacementID: "zone_0",
				CampaignID:  "camp-2",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toPlacementReportRowDTO(tt.row, 0)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestToKeywordReportRowDTO(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		row  reportMetricsCHRow
		want KeywordReportRowDTO
	}{
		{
			name: "profit and roi",
			row: reportMetricsCHRow{
				Dimension:    "insurance",
				CampaignID:   "camp-1",
				Impressions:  5000,
				Clicks:       200,
				Conversions:  5,
				SpendMicro:   25_000_000,
				RevenueMicro: 30_000_000,
			},
			want: KeywordReportRowDTO{
				Keyword:      "insurance",
				CampaignID:   "camp-1",
				Impressions:  5000,
				Clicks:       200,
				Conversions:  5,
				SpendMicro:   25_000_000,
				RevenueMicro: 30_000_000,
				ProfitMicro:  5_000_000,
				ROIPct:       20,
				CPAMicro:     5_000_000,
				CTR:          0.04,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toKeywordReportRowDTO(tt.row, 0)
			assert.Equal(t, tt.want, got)
		})
	}
}
