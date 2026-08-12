package payment

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bidshard/ad-event-processor/internal/config"
	"github.com/bidshard/ad-event-processor/pkg/coldpath"

	"github.com/stretchr/testify/require"
)

func TestColdPathJSON_StripeWebhookRejectsOversizeBody(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{}
	cfg.StripeWebhookSecret = "whsec_test"
	h := NewWebhookHandler(&Service{}, cfg)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	body := strings.Repeat("x", coldpath.PaymentWebhookMaxBody+1)
	req := httptest.NewRequest(http.MethodPost, "/webhooks/stripe", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}
