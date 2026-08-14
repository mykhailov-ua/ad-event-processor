package adminapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStubRoutes_return501(t *testing.T) {
	// Guard: when stubRouteCatalog is non-empty, every entry must return 501 NOT_IMPLEMENTED.
	t.Parallel()
	mux := http.NewServeMux()
	(&StubHTTPHandlers{}).Register(mux)

	for _, route := range stubRouteCatalog {
		req := httptest.NewRequest(route.Method, route.Path, http.NoBody)
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
