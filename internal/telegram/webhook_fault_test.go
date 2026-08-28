package telegram

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestFault_TelegramWebhook_ackUnder500ms(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	(&HTTPHandlers{Telegram: ServiceStub{}}).Register(mux)

	body := []byte(`{"update_id":1,"message":{"chat":{"id":1,"type":"private"},"text":"/start token","from":{"id":42,"is_premium":false}}}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/telegram/webhook/1", bytes.NewReader(body))
	req.Header.Set("X-Telegram-Bot-Api-Secret-Token", "secret")
	rec := httptest.NewRecorder()

	start := time.Now()
	mux.ServeHTTP(rec, req)
	elapsed := time.Since(start)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Less(t, elapsed, 500*time.Millisecond)
	t.Log("fault_proof fault=tg_webhook_ack_budget_under_500ms")
}
