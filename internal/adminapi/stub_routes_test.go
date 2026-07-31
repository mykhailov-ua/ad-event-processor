package adminapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"espx/internal/openapi"
)

func TestStubCatalog_matchesOpenAPI(t *testing.T) {
	t.Parallel()
	if len(stubRouteCatalog) != len(openapi.StubRoutes) {
		t.Fatalf("catalog=%d openapi=%d", len(stubRouteCatalog), len(openapi.StubRoutes))
	}
	for _, route := range stubRouteCatalog {
		if !openapi.IsStub(route.Method, route.Path) {
			t.Fatalf("missing openapi stub: %s %s", route.Method, route.Path)
		}
	}
}

func TestStubRoutes_return501(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	(&StubHTTPHandlers{}).Register(mux)

	for _, route := range stubRouteCatalog {
		req := httptest.NewRequest(route.Method, route.Path, nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusNotImplemented {
			t.Fatalf("%s %s status=%d body=%s", route.Method, route.Path, rec.Code, rec.Body.String())
		}
		var body struct {
			Error struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("%s %s decode: %v", route.Method, route.Path, err)
		}
		if body.Error.Code != "NOT_IMPLEMENTED" || body.Error.Message != stubNotImplementedMessage {
			t.Fatalf("%s %s body=%+v", route.Method, route.Path, body)
		}
	}
}
