package controlplane

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
)

type stubFeedbackRecorder struct {
	meta    SupportFeedbackMeta
	last    SupportFeedbackRecord
	created uuid.UUID
}

func (s *stubFeedbackRecorder) SupportFeedbackMeta(context.Context) (SupportFeedbackMeta, error) {
	return s.meta, nil
}

func (s *stubFeedbackRecorder) RecordSupportFeedback(_ context.Context, in SupportFeedbackRecord) (uuid.UUID, error) {
	s.last = in
	if s.created == uuid.Nil {
		s.created = uuid.New()
	}
	return s.created, nil
}

func TestSupportFeedbackMeta_handler(t *testing.T) {
	t.Parallel()
	rec := &stubFeedbackRecorder{
		meta: SupportFeedbackMeta{
			DeploymentID:  "dep-1",
			BinaryVersion: "1.2.3",
		},
	}
	h := &SupportHTTPHandlers{Feedback: rec}
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/support/feedback/meta", http.NoBody)
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var got SupportFeedbackMeta
	if err := json.Unmarshal(recorder.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.DeploymentID != "dep-1" || got.BinaryVersion != "1.2.3" {
		t.Fatalf("meta=%+v", got)
	}
}

func TestSupportFeedbackPost_handler(t *testing.T) {
	t.Parallel()
	feedback := &stubFeedbackRecorder{
		meta: SupportFeedbackMeta{BinaryVersion: "dev"},
	}
	h := &SupportHTTPHandlers{Feedback: feedback}
	mux := http.NewServeMux()
	h.Register(mux)

	body := `{"type":"bug","contact_email":"ops@example.com","message":"slow dashboard","attach_bundle":false}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/support/feedback", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if feedback.last.ContactEmail != "ops@example.com" || feedback.last.Type != "bug" {
		t.Fatalf("record=%+v", feedback.last)
	}
	if len(feedback.last.BundleGzip) != 0 {
		t.Fatalf("expected no bundle, got %d bytes", len(feedback.last.BundleGzip))
	}
}

func TestSupportFeedbackPost_bundleRedaction(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	raw, err := os.ReadFile(filepath.Join("..", "..", "pkg", "supportbundle", "testdata", "bundle_sample.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "app.log"), raw, 0o644); err != nil {
		t.Fatal(err)
	}

	feedback := &stubFeedbackRecorder{meta: SupportFeedbackMeta{BinaryVersion: "dev"}}
	h := &SupportHTTPHandlers{
		Feedback:      feedback,
		SupportBundle: stubBundler{logDir: dir},
	}
	mux := http.NewServeMux()
	h.Register(mux)

	body := `{"type":"support","contact_email":"ops@example.com","message":"need help","attach_bundle":true}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/support/feedback", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if len(feedback.last.BundleGzip) < 2 {
		t.Fatal("expected gzip bundle")
	}
	if err := validateFeedbackBundleRedaction(feedback.last.BundleGzip); err != nil {
		t.Fatalf("bundle redaction: %v", err)
	}
	logBody, err := readBundleLog(feedback.last.BundleGzip)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(logBody), "203.0.113.10") || strings.Contains(string(logBody), "sk_live") {
		t.Fatalf("secrets in bundle log: %s", logBody)
	}
}

func readBundleLog(bundle []byte) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewReader(bundle))
	if err != nil {
		return nil, err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if hdr.Name == "logs/redacted.log" {
			return io.ReadAll(tr)
		}
	}
	return nil, io.ErrUnexpectedEOF
}
