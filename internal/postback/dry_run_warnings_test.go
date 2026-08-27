package postback

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
)

func TestDryRunConfig_MetaFixtureSkipsClickIdWarnings(t *testing.T) {
	res := DryRunConfig(t.Context(), "facebook", "123456789", "token", "conversion", "", uuid.New())
	if res.OK {
		t.Fatalf("expected dry-run failure without reachable graph, got ok=%v err=%q", res.OK, res.Error)
	}
	if len(res.Warnings) != 0 {
		t.Fatalf("dry-run fixture populates network ids; warnings = %#v", res.Warnings)
	}
}

func TestDryRunConfig_webhookOkWithoutClickIdWarnings(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := DryRunConfig(t.Context(), "webhook", srv.URL, "", "conversion", "", uuid.New())
	if !res.OK {
		t.Fatalf("ok=false err=%q", res.Error)
	}
	if len(res.Warnings) != 0 {
		t.Fatalf("warnings = %#v", res.Warnings)
	}
}
