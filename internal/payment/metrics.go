package payment

import (
	checkout "ad-event-processor/internal/payment/checkout"
	settlement "ad-event-processor/internal/payment/settlement"
	webhook "ad-event-processor/internal/payment/webhook"
)

var (
	IntentsTotal                  = checkout.IntentsTotal
	WebhookEventsTotal            = webhook.WebhookEventsTotal
	WebhookSignatureFailuresTotal = webhook.WebhookSignatureFailuresTotal
	OutboxPending                 = settlement.OutboxPending
	SettlementErrorsTotal         = settlement.SettlementErrorsTotal
	FinancialReconRunsTotal       = settlement.FinancialReconRunsTotal
	FinancialReconFindingsTotal   = settlement.FinancialReconFindingsTotal
)
