package trafficoptimizer

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	EvalTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "traffic_optimizer_eval_total",
		Help: "Traffic optimizer rule evaluations by customer and outcome",
	}, []string{"customer_id", "outcome"})

	WeightUpdatesTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "traffic_optimizer_weight_updates_total",
		Help: "Traffic optimizer weight apply operations by scope",
	}, []string{"scope"})

	LastTickSeconds = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "traffic_optimizer_last_tick_seconds",
		Help: "Unix seconds of the last traffic optimizer worker tick",
	})
)
