package webhook

import (
	"testing"

	"ad-event-processor/internal/payment/db"
)

func BenchmarkIsValidTransition(b *testing.B) {
	statuses := []db.PaymentPaymentIntentStatus{
		db.PaymentPaymentIntentStatusCREATED,
		db.PaymentPaymentIntentStatusPENDINGPROVIDER,
		db.PaymentPaymentIntentStatusPROCESSING,
		db.PaymentPaymentIntentStatusSUCCEEDED,
		db.PaymentPaymentIntentStatusFAILED,
	}
	benchN := 0
	for b.Loop() {
		from := statuses[benchN%len(statuses)]
		to := statuses[(benchN+1)%len(statuses)]
		_ = isValidTransition(from, to)
		benchN++
	}
}
