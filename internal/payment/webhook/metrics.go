package webhook

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	WebhookEventsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "payment_webhook_events_total",
		Help: "Stripe webhook events processed by outcome",
	}, []string{"outcome"})

	WebhookSignatureFailuresTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "payment_webhook_signature_failures_total",
		Help: "Rejected Stripe webhook signatures",
	})
)
