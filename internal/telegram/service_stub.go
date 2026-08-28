package telegram

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type ServiceStub struct{}

func (s ServiceStub) ValidateInitData(context.Context, uuid.UUID, string) (ValidateResult, error) {
	return ValidateResult{}, nil
}

func (s ServiceStub) MintClick(context.Context, uuid.UUID) (ClickMintResult, error) {
	return ClickMintResult{}, nil
}

func (s ServiceStub) ReceiveWebhook(context.Context, int64, string, []byte) error {
	return nil
}

func (s ServiceStub) CreateDeeplink(context.Context, DeeplinkDTO) (DeeplinkDTO, error) {
	return DeeplinkDTO{}, nil
}

func (s ServiceStub) GetDeeplink(context.Context, string) (DeeplinkDTO, error) {
	return DeeplinkDTO{}, nil
}

func (s ServiceStub) ConfigureBot(context.Context, BotDTO) error { return nil }
func (s ServiceStub) ListBots(context.Context) ([]BotDTO, error) { return nil, nil }
func (s ServiceStub) GetBot(context.Context, uuid.UUID) (BotDTO, error) {
	return BotDTO{}, nil
}
func (s ServiceStub) CreatePostback(context.Context, PostbackDTO) error { return nil }
func (s ServiceStub) UpdatePostback(context.Context, uuid.UUID, string) error {
	return nil
}

func (s ServiceStub) GetPostback(context.Context, uuid.UUID) (PostbackDTO, error) {
	return PostbackDTO{}, nil
}

func (s ServiceStub) ListPostbacks(context.Context, uuid.UUID) ([]PostbackDTO, error) {
	return nil, nil
}
func (s ServiceStub) DeletePostback(context.Context, uuid.UUID) error { return nil }
func (s ServiceStub) TestPostback(context.Context, uuid.UUID) error   { return nil }
func (s ServiceStub) GetTelegramReport(context.Context, time.Time, time.Time, ReportFilter) ([]byte, error) {
	return nil, nil
}

func (s ServiceStub) GetTelegramSummaryReport(context.Context, time.Time, time.Time, ReportFilter) ([]byte, error) {
	return nil, nil
}

func (s ServiceStub) GetTelegramFunnelReport(context.Context, time.Time, time.Time, ReportFilter) ([]byte, error) {
	return nil, nil
}

func (s ServiceStub) GetTelegramBotsReport(context.Context, time.Time, time.Time, ReportFilter) ([]byte, error) {
	return nil, nil
}

func (s ServiceStub) GetTelegramPremiumReport(context.Context, time.Time, time.Time, ReportFilter) ([]byte, error) {
	return nil, nil
}

func (s ServiceStub) GetTelegramFraudReport(context.Context, time.Time, time.Time, ReportFilter) ([]byte, error) {
	return nil, nil
}
