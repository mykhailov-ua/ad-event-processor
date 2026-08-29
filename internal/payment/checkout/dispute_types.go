package checkout

import "ad-event-processor/internal/payment/db"

type DisputeListItem struct {
	Intent            db.PaymentPaymentIntent
	ProviderDisputeID string
}
