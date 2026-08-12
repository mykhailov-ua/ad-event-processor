package adminapi

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type telegramServiceStub struct{}

func (telegramServiceStub) ValidateInitData(context.Context, uuid.UUID, string) (ValidateResult, error) {
	return ValidateResult{}, nil
}
func (telegramServiceStub) MintClick(context.Context, uuid.UUID) (ClickMintResult, error) {
	return ClickMintResult{}, nil
}
func (telegramServiceStub) ReceiveWebhook(context.Context, int64, string, []byte) error {
	return nil
}
func (telegramServiceStub) CreateDeeplink(context.Context, DeeplinkDTO) (DeeplinkDTO, error) {
	return DeeplinkDTO{}, nil
}
func (telegramServiceStub) GetDeeplink(context.Context, string) (DeeplinkDTO, error) {
	return DeeplinkDTO{}, nil
}
func (telegramServiceStub) ConfigureBot(context.Context, BotDTO) error { return nil }
func (telegramServiceStub) ListBots(context.Context) ([]BotDTO, error)   { return nil, nil }
func (telegramServiceStub) GetBot(context.Context, uuid.UUID) (BotDTO, error) {
	return BotDTO{}, nil
}
func (telegramServiceStub) CreatePostback(context.Context, PostbackDTO) error { return nil }
func (telegramServiceStub) UpdatePostback(context.Context, uuid.UUID, string) error {
	return nil
}
func (telegramServiceStub) GetPostback(context.Context, uuid.UUID) (PostbackDTO, error) {
	return PostbackDTO{}, nil
}
func (telegramServiceStub) ListPostbacks(context.Context, uuid.UUID) ([]PostbackDTO, error) {
	return nil, nil
}
func (telegramServiceStub) DeletePostback(context.Context, uuid.UUID) error { return nil }
func (telegramServiceStub) TestPostback(context.Context, uuid.UUID) error   { return nil }
func (telegramServiceStub) GetTelegramReport(context.Context, time.Time, time.Time, TelegramReportFilter) ([]byte, error) {
	return nil, nil
}
func (telegramServiceStub) GetTelegramSummaryReport(context.Context, time.Time, time.Time, TelegramReportFilter) ([]byte, error) {
	return nil, nil
}
func (telegramServiceStub) GetTelegramFunnelReport(context.Context, time.Time, time.Time, TelegramReportFilter) ([]byte, error) {
	return nil, nil
}
func (telegramServiceStub) GetTelegramBotsReport(context.Context, time.Time, time.Time, TelegramReportFilter) ([]byte, error) {
	return nil, nil
}
func (telegramServiceStub) GetTelegramPremiumReport(context.Context, time.Time, time.Time, TelegramReportFilter) ([]byte, error) {
	return nil, nil
}
func (telegramServiceStub) GetTelegramFraudReport(context.Context, time.Time, time.Time, TelegramReportFilter) ([]byte, error) {
	return nil, nil
}

func TestFault_TelegramWebhook_ackUnder500ms(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	(&TelegramHTTPHandlers{Telegram: telegramServiceStub{}}).Register(mux)

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
