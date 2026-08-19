package payment

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/bidshard/ad-event-processor/internal/config"

	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/checkout/session"
)

type ParsedEvent struct {
	EventID         string
	EventType       string
	PaymentIntentID string
	AmountMicro     int64
	ProviderRef     string
}

func StripeConfigured(cfg *config.Config) bool {
	return cfg != nil && string(cfg.StripeSecretKey) != ""
}

func CryptoConfigured(cfg *config.Config) bool {
	return cfg != nil && string(cfg.CryptoWebhookSecret) != ""
}

func DefaultCheckoutProvider(cfg *config.Config) string {
	return "stripe"
}

func LogProviderMode(cfg *config.Config) {
	if StripeConfigured(cfg) {
		slog.Info("payment provider mode", "provider", "stripe", "checkout_api", "stripe_go")
		if string(cfg.StripeWebhookSecret) == "" {
			slog.Warn("STRIPE_WEBHOOK_SECRET unset; POST /webhooks/stripe returns 503")
		}
		if string(cfg.PaymentInternalToken) == "" {
			slog.Warn("PAYMENT_INTERNAL_TOKEN unset; gRPC CreatePaymentIntent rejects callers")
		}
	} else {
		slog.Info("payment provider mode", "provider", "mock")
	}
	if CryptoConfigured(cfg) {
		slog.Info("payment crypto webhook enabled", "provider", "crypto", "webhook_path", "/webhooks/crypto")
	} else {
		slog.Warn("CRYPTO_WEBHOOK_SECRET unset; POST /webhooks/crypto returns 503")
	}
}

func CreateCheckout(
	ctx context.Context,
	cfg *config.Config,
	provider string,
	amountMicro int64,
	currency string,
	metadata map[string]string,
	idempotencyKey string,
) (providerRef, checkoutURL string, err error) {
	switch provider {
	case "crypto":
		prov := CryptoProviderGeneric
		if metadata != nil {
			if p := metadata["crypto_provider"]; p != "" {
				prov = p
			}
		}
		result, err := CreateCryptoCheckout(cfg, prov, amountMicro, idempotencyKey)
		if err != nil {
			return "", "", err
		}
		if metadata != nil {
			metadata["deposit_address"] = result.DepositAddress
			metadata["deposit_network"] = result.DepositNetwork
			metadata["deposit_qr_svg"] = result.DepositQRSVG
		}
		return result.ProviderRef, result.CheckoutURL, nil
	case "stripe", "":
		if StripeConfigured(cfg) {
			return createStripeCheckout(ctx, cfg, amountMicro, currency, metadata, idempotencyKey)
		}
		return createMockCheckout(idempotencyKey)
	default:
		return "", "", fmt.Errorf("unsupported payment provider: %s", provider)
	}
}

func createMockCheckout(idempotencyKey string) (providerRef, checkoutURL string, err error) {
	return "pi_mock_" + idempotencyKey, "https://checkout.stripe.dev/pay/mock_" + idempotencyKey, nil
}

func createStripeCheckout(
	ctx context.Context,
	cfg *config.Config,
	amountMicro int64,
	currency string,
	metadata map[string]string,
	idempotencyKey string,
) (providerRef, checkoutURL string, err error) {
	secretKey := string(cfg.StripeSecretKey)
	if secretKey == "" {
		return "", "", ErrProviderNotConfigured
	}
	if _, err := MicroToStripeAmount(amountMicro); err != nil {
		return "", "", fmt.Errorf("stripe checkout amount: %w", err)
	}
	_ = ctx
	return createStripeCheckoutSession(secretKey, cfg.StripeCheckoutSuccessURL, cfg.StripeCheckoutCancelURL, amountMicro, currency, metadata, idempotencyKey)
}

func createStripeCheckoutSession(secretKey, successURL, cancelURL string, amountMicro int64, currency string, metadata map[string]string, idempotencyKey string) (providerRef, checkoutURL string, err error) {
	cents, err := MicroToStripeAmount(amountMicro)
	if err != nil {
		return "", "", err
	}
	cur := strings.ToLower(strings.TrimSpace(currency))
	if cur == "" {
		cur = "usd"
	}
	if successURL == "" || cancelURL == "" {
		return "", "", ErrProviderNotConfigured
	}

	stripe.Key = secretKey

	piData := &stripe.CheckoutSessionPaymentIntentDataParams{}
	for k, v := range metadata {
		if k != "" && v != "" {
			piData.AddMetadata(k, v)
		}
	}

	sessionParams := &stripe.CheckoutSessionParams{
		Mode:       stripe.String(string(stripe.CheckoutSessionModePayment)),
		SuccessURL: stripe.String(successURL),
		CancelURL:  stripe.String(cancelURL),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
					Currency: stripe.String(cur),
					ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
						Name: stripe.String("Account balance top-up"),
					},
					UnitAmount: stripe.Int64(cents),
				},
				Quantity: stripe.Int64(1),
			},
		},
		PaymentIntentData: piData,
	}
	sessionParams.SetIdempotencyKey("cs_" + idempotencyKey)
	sessionParams.AddExpand("payment_intent")

	sess, err := session.New(sessionParams)
	if err != nil {
		return "", "", fmt.Errorf("stripe checkout session: %w", err)
	}
	if sess.URL == "" {
		return "", "", fmt.Errorf("stripe checkout session missing url")
	}
	piID := ""
	if sess.PaymentIntent != nil {
		piID = sess.PaymentIntent.ID
	}
	if piID == "" {
		return "", "", fmt.Errorf("stripe checkout session missing payment_intent")
	}
	return piID, sess.URL, nil
}
