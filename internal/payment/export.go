package payment

import (
	"context"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/notify"
	checkout "ad-event-processor/internal/payment/checkout"
	"ad-event-processor/internal/payment/db"
	settlement "ad-event-processor/internal/payment/settlement"
	webhook "ad-event-processor/internal/payment/webhook"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type (
	Handler                 = checkout.Handler
	APIClient               = checkout.APIClient
	OutboxWorker            = settlement.OutboxWorker
	CryptoHoldWorker        = settlement.CryptoHoldWorker
	ReconService            = settlement.ReconService
	NotifierClient          = settlement.NotifierClient
	SettlementHandler       = settlement.SettlementHandler
	SettlementHost          = settlement.SettlementHost
	CTVSettlementHost       = settlement.CTVSettlementHost
	SettlementLedgerClient  = settlement.SettlementLedgerClient
	SettlementBatchParams   = settlement.SettlementBatchParams
	SettlementBatchResult   = settlement.SettlementBatchResult
	SettlementCreditParams  = settlement.SettlementCreditParams
	FinancialReconAlerter   = settlement.FinancialReconAlerter
	SettlementFailedAlerter = settlement.SettlementFailedAlerter
	WebhookHandler          = webhook.WebhookHandler
	DisputeListItem         = checkout.DisputeListItem
)

const (
	CryptoProviderBTCPay         = checkout.CryptoProviderBTCPay
	CryptoProviderCryptomus      = checkout.CryptoProviderCryptomus
	OutboxEventReverseBalance    = settlement.OutboxEventReverseBalance
	OutboxEventApplyChargeback   = settlement.OutboxEventApplyChargeback
	OutboxEventReverseChargeback = settlement.OutboxEventReverseChargeback
)

func StripeConfigured(cfg *config.Config) bool {
	return checkout.StripeConfigured(cfg)
}

func CryptoConfigured(cfg *config.Config) bool {
	return checkout.CryptoConfigured(cfg)
}

func DefaultCheckoutProvider(cfg *config.Config) string {
	return checkout.DefaultCheckoutProvider(cfg)
}

func LogProviderMode(cfg *config.Config) {
	checkout.LogProviderMode(cfg)
}

func MicroToStripeAmount(amountMicro int64) (int64, error) {
	return checkout.MicroToStripeAmount(amountMicro)
}

func StripeAmountToMicro(stripeAmount int64) int64 {
	return checkout.StripeAmountToMicro(stripeAmount)
}

func CreateCryptoCheckout(cfg *config.Config, provider string, amountMicro int64, idempotencyKey string) (checkout.CryptoCheckoutResult, error) {
	return checkout.CreateCryptoCheckout(cfg, provider, amountMicro, idempotencyKey)
}

func CreateCheckout(ctx context.Context, cfg *config.Config, providerName string, amountMicro int64, currency string, metadata map[string]string, idempotencyKey string) (providerRef string, checkoutURL string, err error) {
	return checkout.CreateCheckout(ctx, cfg, providerName, amountMicro, currency, metadata, idempotencyKey)
}

func NormalizeCryptoProvider(name string) string {
	return checkout.NormalizeCryptoProvider(name)
}

func SignBTCPayWebhookBody(body []byte, secret string) string {
	return checkout.SignBTCPayWebhookBody(body, secret)
}

func VerifyBTCPayWebhookSignature(body []byte, sig, secret string) bool {
	return checkout.VerifyBTCPayWebhookSignature(body, sig, secret)
}

func SignCryptomusWebhookFields(fields map[string]any, apiKey string) ([]byte, string, error) {
	return checkout.SignCryptomusWebhookFields(fields, apiKey)
}

func VerifyCryptomusWebhookSignature(body []byte, sign, apiKey string) bool {
	return checkout.VerifyCryptomusWebhookSignature(body, sign, apiKey)
}

func NewHandler(svc *Service, cfg *config.Config) *Handler {
	return checkout.NewHandler(svc.checkout, svc.webhook, cfg)
}

func NewOutboxWorker(pool *pgxpool.Pool, cfg *config.Config) *OutboxWorker {
	return settlement.NewOutboxWorker(pool, cfg)
}

func NewCryptoHoldWorker(pool *pgxpool.Pool, cfg *config.Config) *CryptoHoldWorker {
	return settlement.NewCryptoHoldWorker(pool, cfg)
}

func NewReconService(paymentPool *pgxpool.Pool, ledger *SettlementLedgerClient, alerter *FinancialReconAlerter) *ReconService {
	return settlement.NewReconService(paymentPool, ledger, alerter)
}

func NewSettlementLedgerClient(cfg *config.Config) *SettlementLedgerClient {
	return settlement.NewSettlementLedgerClient(cfg)
}

func NewSettlementHandler(host SettlementHost, cfg *config.Config) *SettlementHandler {
	return settlement.NewSettlementHandler(host, cfg)
}

func ApplyCTVSettlement(
	ctx context.Context,
	host CTVSettlementHost,
	settlementID string,
	customerID, campaignID uuid.UUID,
	spendMicro int64,
) (domain.CTVSettlementResult, error) {
	return settlement.ApplyCTVSettlement(ctx, host, settlementID, customerID, campaignID, spendMicro)
}

func NewAPIClientFromAPI(api domain.PaymentAPI, token string) *APIClient {
	return checkout.NewAPIClientFromAPI(api, token)
}

func NewAPIClientInProcess(api domain.PaymentAPI, token string) *APIClient {
	return checkout.NewAPIClientInProcess(api, token)
}

func NewSettlementFailedAlerter(client *NotifierClient, cfg *config.Config) *SettlementFailedAlerter {
	return settlement.NewSettlementFailedAlerter(client, cfg)
}

func NewFinancialReconAlerter(client *NotifierClient, cfg *config.Config) *FinancialReconAlerter {
	return settlement.NewFinancialReconAlerter(client, cfg)
}

func NewInProcessNotifierClient(api notify.NotifierAPI) *NotifierClient {
	return settlement.NewInProcessNotifierClient(api)
}

func ResolveNotifierClient(ctx context.Context, cfg *config.Config) (*NotifierClient, func(), error) {
	return settlement.ResolveNotifierClient(ctx, cfg)
}

func NewWebhookHandler(svc *Service, cfg *config.Config) *WebhookHandler {
	return webhook.NewWebhookHandler(svc.webhook, cfg)
}

func SetPostSettlementMarkHookForTest(hook func(context.Context, db.PaymentPaymentOutbox) error) {
	settlement.SetPostSettlementMarkHookForTest(hook)
}
