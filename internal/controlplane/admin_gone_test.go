package controlplane

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRootReturnsJSONNotFound(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	registerRootRoute(mux, nil)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d want %d body=%s", rec.Code, http.StatusNotFound, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("content-type=%q want application/json", ct)
	}
	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v body=%s", err, rec.Body.String())
	}
	if body.Error.Code != "NOT_FOUND" {
		t.Fatalf("code=%q want NOT_FOUND", body.Error.Code)
	}
}
