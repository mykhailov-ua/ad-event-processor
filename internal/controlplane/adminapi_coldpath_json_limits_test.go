package controlplane

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"ad-event-processor/internal/campaign/selfserve"
	"ad-event-processor/internal/opsadmin"
	"ad-event-processor/pkg/coldpath"

	"github.com/stretchr/testify/require"
)

type stubConsentRecorder struct{}

func (r stubConsentRecorder) RecordConsent(_ context.Context, _ opsadmin.ConsentRecord) error {
	return nil
}

type stubConsentVerifier struct{}

func (v stubConsentVerifier) Verify(_ []byte, _ string) error {
	return nil
}

func TestColdPathJSON_SelfServePaymentIntentRejectsOversizeBody(t *testing.T) {
	t.Parallel()

	h := &selfserve.SelfServeHTTPHandlers{}
	mux := http.NewServeMux()
	h.Register(mux)

	body := strings.Repeat("x", coldpath.SelfServePaymentIntentMaxBody+1)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/selfserve/payment-intents", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestColdPathJSON_ConsentRejectsOversizeBody(t *testing.T) {
	t.Parallel()

	h := &opsadmin.HTTPHandlers{
		ConsentRecorder: stubConsentRecorder{},
		ConsentVerifier: stubConsentVerifier{},
	}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/consent", h.PostConsent)

	body := strings.Repeat("x", coldpath.DefaultMaxBody+1)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/consent", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}
