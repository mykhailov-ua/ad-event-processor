package features

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	residentialIntelFeedAppendedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "residential_intel_feed_appended_total",
		Help: "External residential intel feed lines appended by cold enricher",
	})

	residentialIntelLookupsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "residential_intel_provider_lookups_total",
		Help: "External residential intel provider lookups (cold path)",
	})

	residentialIntelErrorsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "residential_intel_errors_total",
		Help: "Residential intel enricher errors",
	})
)
