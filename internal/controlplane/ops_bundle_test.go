package controlplane

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"ad-event-processor/pkg/supportbundle"
)

type stubBundler struct {
	logDir string
}

func (s stubBundler) WriteSupportBundle(ctx context.Context, w io.Writer) error {
	return supportbundle.Write(ctx, w, supportbundle.Options{
		Meta:   supportbundle.Meta{DeploymentID: "test-dep", LicenseState: "ACTIVE"},
		LogDir: s.logDir,
	})
}

func TestBundleRedaction_handler(t *testing.T) {
	t.Parallel()
	h := &OpsHTTPHandlers{
		SupportBundle: stubBundler{logDir: t.TempDir()},
	}
	mux := http.NewServeMux()
	h.registerSupportBundleRoutes(mux)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/ops/support/bundle", http.NoBody)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/gzip" {
		t.Fatalf("content-type=%q", ct)
	}
	body := rec.Body.Bytes()
	if len(body) < 2 || body[0] != 0x1f || body[1] != 0x8b {
		t.Fatalf("not gzip payload: %d bytes", len(body))
	}
}
