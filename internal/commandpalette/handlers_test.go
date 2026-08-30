package commandpalette

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"ad-event-processor/internal/controlplane/authz"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPHandlers_search_returnsCatalogWhenStoreEmpty(t *testing.T) {
	t.Parallel()
	custID := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	h := testHandlers(custID)
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/command-palette/search?customer_id="+custID.String()+"&q=integrations", http.NoBody)
	req = req.WithContext(authz.WithSnapshot(req.Context(), fullReadSnapshot()))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var resp SearchResponse
	require.NoError(t, decodeJSON(rec.Body.Bytes(), &resp))
	require.NotEmpty(t, resp.Items)
	found := false
	for _, item := range resp.Items {
		if item.Href == "/integrations" {
			found = true
			assert.Equal(t, "route", item.Kind)
		}
	}
	assert.True(t, found)
}

func TestHTTPHandlers_search_returnsStoreResults(t *testing.T) {
	t.Parallel()
	custID := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	searcher := &recordingSearcher{
		items: []ItemDTO{{
			ID:    "00000000-0000-4000-8000-000000000099",
			Kind:  "campaign",
			Label: "Camp Alpha",
			Href:  "/campaigns/00000000-0000-4000-8000-000000000099",
			Group: "campaigns",
		}},
	}
	h := testHandlersWithSearcher(custID, searcher)
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/command-palette/search?customer_id="+custID.String()+"&q=camp", http.NoBody)
	req = req.WithContext(authz.WithSnapshot(req.Context(), fullReadSnapshot()))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var resp SearchResponse
	require.NoError(t, decodeJSON(rec.Body.Bytes(), &resp))
	require.NotEmpty(t, resp.Items)
	var foundCampaign bool
	for _, item := range resp.Items {
		if item.ID == "00000000-0000-4000-8000-000000000099" {
			foundCampaign = true
			assert.Equal(t, "Camp Alpha", item.Label)
		}
	}
	assert.True(t, foundCampaign)
	assert.GreaterOrEqual(t, resp.Total, int64(1))
	assert.Equal(t, 25, resp.Limit)
	assert.False(t, resp.Degraded)
	assert.True(t, searcher.called)
}

func TestHTTPHandlers_search_emptyQuerySkipsStore(t *testing.T) {
	t.Parallel()
	custID := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	searcher := &recordingSearcher{}
	h := testHandlersWithSearcher(custID, searcher)
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/command-palette/search?customer_id="+custID.String(), http.NoBody)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var resp SearchResponse
	require.NoError(t, decodeJSON(rec.Body.Bytes(), &resp))
	assert.Empty(t, resp.Items)
	assert.False(t, searcher.called)
}

func TestHTTPHandlers_search_queryTooShort(t *testing.T) {
	t.Parallel()
	custID := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	h := testHandlers(custID)
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/command-palette/search?customer_id="+custID.String()+"&q=a", http.NoBody)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestCommandPalette_maxQueryLen_holdout(t *testing.T) {
	t.Parallel()
	custID := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	h := testHandlers(custID)
	h.MaxQueryLen = 8
	mux := http.NewServeMux()
	h.Register(mux)

	longQ := strings.Repeat("a", 9)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/command-palette/search?customer_id="+custID.String()+"&q="+longQ, http.NoBody)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestCommandPalette_degradedWithCatalog_holdout(t *testing.T) {
	t.Parallel()
	custID := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	svc := &Service{Store: failingEntitySearcher{}}
	ctx := authz.WithSnapshot(t.Context(), fullReadSnapshot())
	resp := svc.Search(ctx, custID, "integrations", DefaultSearchLimit, nil, func(string) bool { return true })
	require.True(t, resp.Degraded)
	require.NotEmpty(t, resp.Items)
	found := false
	for _, item := range resp.Items {
		if item.Href == "/integrations" {
			found = true
			break
		}
	}
	require.True(t, found, "expected catalog route in degraded search response")
}

func TestHTTPHandlers_listRoutes_returnsNavEntries(t *testing.T) {
	t.Parallel()
	custID := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	h := testHandlers(custID)
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/command-palette/routes", http.NoBody)
	req = req.WithContext(authz.WithSnapshot(req.Context(), authz.Snapshot{
		Permissions: map[string]struct{}{
			authz.PermCampaignsRead: {},
			"billing:read":          {},
			"customers:read":        {},
			"ops:read":              {},
			"rtb:read":              {},
		},
		Mask: authz.MaskFull,
	}))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var resp RoutesResponse
	require.NoError(t, decodeJSON(rec.Body.Bytes(), &resp))
	require.NotEmpty(t, resp.Items)
	assert.GreaterOrEqual(t, resp.Total, len(RequiredLiveNavPaths))
}

func TestCommandPalette_routesCatalog_holdout(t *testing.T) {
	t.Parallel()
	hrefs := catalogHrefSet(NavCatalogEntries())
	for _, path := range RequiredLiveNavPaths {
		_, ok := hrefs[path]
		assert.True(t, ok, "required live nav path %q missing from command palette catalog", path)
	}
}

func TestCommandPalette_licenseGatedRoute_holdout(t *testing.T) {
	t.Parallel()
	ctx := authz.WithSnapshot(t.Context(), authz.Snapshot{
		Permissions: map[string]struct{}{
			authz.PermCampaignsRead: {},
			"rtb:read":              {},
		},
		Mask: authz.MaskFull,
	})
	items := FilterNavCatalog(ctx, NavCatalogEntries(), func(featureKey string) bool {
		return featureKey != "openrtb"
	})
	for _, item := range items {
		assert.NotEqual(t, "/rtb", item.Href)
	}
}

func testHandlers(custID uuid.UUID) *HTTPHandlers {
	return testHandlersWithSearcher(custID, emptySearcher{})
}

func testHandlersWithSearcher(custID uuid.UUID, store EntitySearcher) *HTTPHandlers {
	return &HTTPHandlers{
		Search: &Service{Store: store},
		LicenseFeatureAllowed: func(featureKey string) (bool, string) {
			if featureKey == "openrtb" || featureKey == "ad_platform_campaign_api" {
				return true, "scale"
			}
			return true, "pro"
		},
		ResolveCustomerID: func(_ *http.Request, _ *uuid.UUID) (uuid.UUID, error) {
			return custID, nil
		},
	}
}

func decodeJSON(data []byte, v any) error {
	return json.Unmarshal(data, v)
}

type emptySearcher struct{}

func (emptySearcher) SearchEntities(context.Context, uuid.UUID, string, int, []string) ([]ItemDTO, error) {
	return []ItemDTO{}, nil
}

type failingEntitySearcher struct{}

func (failingEntitySearcher) SearchEntities(context.Context, uuid.UUID, string, int, []string) ([]ItemDTO, error) {
	return nil, errors.New("postgres unavailable")
}
