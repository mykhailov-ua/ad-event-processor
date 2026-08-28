package platformadmin

import (
	"context"
	"time"

	"ad-event-processor/internal/metrics"
)

type VendorTelemetryHost interface {
	VendorTelemetryEnabled() bool
	VendorTelemetryInterval() time.Duration
	VendorTelemetryTimeout() time.Duration
	GeoIPDBPath() string
	StripeSecretKey() string
	TelegramBotToken() string
	SMTPHost() string
	SMTPPort() string
	StartWorker(fn func())
	WorkerContext() context.Context
}

type vendorProbeMetrics struct{}

func (m vendorProbeMetrics) ObserveProbe(vendor string, success bool, latency time.Duration) {
	val := 0.0
	if success {
		val = 1.0
	}
	metrics.VendorProbeSuccess.WithLabelValues(vendor).Set(val)
	metrics.VendorProbeLatencySeconds.WithLabelValues(vendor).Observe(latency.Seconds())
}

func (m vendorProbeMetrics) ObserveProbeError(vendor string) {
	metrics.VendorProbeErrorsTotal.WithLabelValues(vendor).Inc()
}

func StartVendorTelemetryWorker(host VendorTelemetryHost) {
	if host == nil || !host.VendorTelemetryEnabled() {
		return
	}
	opts := Options{
		GeoIPDBPath:      host.GeoIPDBPath(),
		StripeSecretKey:  host.StripeSecretKey(),
		TelegramBotToken: host.TelegramBotToken(),
		SMTPHost:         host.SMTPHost(),
		SMTPPort:         host.SMTPPort(),
	}
	reg := RegistryFromOptions(opts)
	worker := NewWorker(reg, WorkerConfig{
		Interval: host.VendorTelemetryInterval(),
		Timeout:  host.VendorTelemetryTimeout(),
	}, vendorProbeMetrics{})
	host.StartWorker(func() {
		worker.Start(host.WorkerContext())
	})
}
