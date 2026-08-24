package payment

import (
	"ad-event-processor/internal/payment/db"
	"ad-event-processor/pkg/coldpath"
)

const metadataCheckoutURLKey = "checkout_url"

func mergeIntentMetadata(base map[string]string, checkoutURL string) ([]byte, error) {
	meta := make(map[string]string, len(base)+1)
	for k, v := range base {
		meta[k] = v
	}
	if checkoutURL != "" {
		meta[metadataCheckoutURLKey] = checkoutURL
	}
	return coldpath.MarshalJSON(meta)
}

func checkoutURLFromIntent(intent db.PaymentPaymentIntent) string {
	var meta map[string]string
	if err := coldpath.UnmarshalJSON(intent.Metadata, &meta); err != nil {
		return ""
	}
	return meta[metadataCheckoutURLKey]
}
