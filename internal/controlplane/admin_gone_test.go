package controlplane

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLegacyAdminRoutesReturnGone(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	registerAdminGoneRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/admin/campaigns", http.NoBody)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusGone {
		t.Fatalf("status=%d want %d body=%s", rec.Code, http.StatusGone, rec.Body.String())
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
	if body.Error.Code != "GONE" {
		t.Fatalf("code=%q want GONE", body.Error.Code)
	}
}
