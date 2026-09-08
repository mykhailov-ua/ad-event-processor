package domains

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var domainSSLRenewTotal = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "ad_event_processor_domain_ssl_renew_total",
	Help: "Wildcard domain SSL renew attempts by outcome",
}, []string{"status"})

func recordDomainSSLRenew(status string) {
	if status == "" {
		status = "unknown"
	}
	domainSSLRenewTotal.WithLabelValues(status).Inc()
}
