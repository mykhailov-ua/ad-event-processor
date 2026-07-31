package ivtdetector

import (
	"espx/internal/fraudscoring"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	ivtCandidatesTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ivt_candidates_total",
		Help: "Suspicious IP candidates discovered per detection rule",
	}, []string{"rule"})

	ivtEnqueuedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ivt_enqueued_total",
		Help: "Blacklist enqueue operations completed by the IVT detector",
	})

	ivtBackpressureDropsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ivt_backpressure_drops_total",
		Help: "Detector cycles skipped due to management outbox backpressure",
	})

	fraudScoringDurationSeconds = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "fraud_scoring_duration_seconds",
		Help:    "Duration of ML scoring in seconds",
		Buckets: prometheus.DefBuckets,
	})

	fraudScoringCandidatesTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "fraud_scoring_candidates_total",
		Help: "Total number of candidates scored by ML",
	})

	fraudScoringErrorsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "fraud_scoring_errors_total",
		Help: "Total number of ML scoring errors",
	})

	fraudEnforcementEnqueuedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "fraud_enforcement_enqueued_total",
		Help: "Total number of ML enforcement threats enqueued",
	}, []string{"action"})

	mlShadowScore = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "ml_shadow_score",
		Help:    "Raw ML probability scores from shadow scoring",
		Buckets: []float64{0.05, 0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 0.95, 1.0},
	})

	mlShadowTierTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ml_shadow_tier_total",
		Help: "Shadow scoring decisions by fraud tier",
	}, []string{"tier"})

	mlShadowActionTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ml_shadow_action_total",
		Help: "Shadow scoring enforcement actions (boost, ghost, blacklist)",
	}, []string{"action"})
)

func recordShadowMetrics(mlScore float64, tier fraudscoring.FraudTier, action string) {
	mlShadowScore.Observe(mlScore)
	mlShadowTierTotal.WithLabelValues(string(tier)).Inc()
	if action != "" {
		mlShadowActionTotal.WithLabelValues(action).Inc()
	}
}
