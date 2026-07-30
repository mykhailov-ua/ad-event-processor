package payment

import (
	"context"
	"fmt"
	"strings"

	"espx/internal/config"

	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/checkout/session"
)

type StripeProvider struct {
	secretKey  string
	successURL string
	cancelURL  string
}

func NewStripeProvider(cfg *config.Config) *StripeProvider {
	return &StripeProvider{
		secretKey:  string(cfg.StripeSecretKey),
		successURL: cfg.StripeCheckoutSuccessURL,
		cancelURL:  cfg.StripeCheckoutCancelURL,
	}
}

func (p *StripeProvider) Name() string {
	return "stripe"
}

func (p *StripeProvider) CreateCheckout(ctx context.Context, amountMicro int64, currency string, metadata map[string]string, idempotencyKey string) (string, string, error) {
	if p.secretKey == "" {
		return "", "", ErrProviderNotConfigured
	}
	if _, err := MicroToStripeAmount(amountMicro); err != nil {
		return "", "", fmt.Errorf("stripe checkout amount: %w", err)
	}
	_ = ctx
	return createStripeCheckoutSession(p.secretKey, p.successURL, p.cancelURL, amountMicro, currency, metadata, idempotencyKey)
}

func createStripeCheckoutSession(secretKey, successURL, cancelURL string, amountMicro int64, currency string, metadata map[string]string, idempotencyKey string) (providerRef string, checkoutURL string, err error) {
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
