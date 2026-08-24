package controlplane

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"ad-event-processor/pkg/coldpath"

	"github.com/stretchr/testify/require"
)

type stubConsentRecorder struct{}

func (stubConsentRecorder) RecordConsent(_ context.Context, _ ConsentRecord) error {
	return nil
}

type stubConsentVerifier struct{}

func (stubConsentVerifier) Verify(_ []byte, _ string) error {
	return nil
}

func TestColdPathJSON_SelfServePaymentIntentRejectsOversizeBody(t *testing.T) {
	t.Parallel()

	h := &SelfServeHTTPHandlers{}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/selfserve/payment-intents", h.createPaymentIntent)

	body := strings.Repeat("x", coldpath.SelfServePaymentIntentMaxBody+1)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/selfserve/payment-intents", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestColdPathJSON_ConsentRejectsOversizeBody(t *testing.T) {
	t.Parallel()

	h := &OpsHTTPHandlers{
		ConsentRecorder: stubConsentRecorder{},
		ConsentVerifier: stubConsentVerifier{},
	}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/consent", h.postConsent)

	body := strings.Repeat("x", coldpath.DefaultMaxBody+1)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/consent", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}
