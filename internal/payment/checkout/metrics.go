package checkout

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var IntentsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "payment_intents_total",
	Help: "Payment intents created or transitioned by status",
}, []string{"status"})
