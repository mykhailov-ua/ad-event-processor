package controlplane

import (
	"time"

	"ad-event-processor/internal/metrics"
	"ad-event-processor/pkg/vendorprobe"
)

type vendorProbeMetrics struct{}

func (vendorProbeMetrics) ObserveProbe(vendor string, success bool, latency time.Duration) {
	val := 0.0
	if success {
		val = 1.0
	}
	metrics.VendorProbeSuccess.WithLabelValues(vendor).Set(val)
	metrics.VendorProbeLatencySeconds.WithLabelValues(vendor).Observe(latency.Seconds())
}

func (vendorProbeMetrics) ObserveProbeError(vendor string) {
	metrics.VendorProbeErrorsTotal.WithLabelValues(vendor).Inc()
}

func (s *Service) StartVendorTelemetryWorker() {
	if s == nil || s.cfg == nil || !s.cfg.VendorTelemetryEnabled {
		return
	}
	opts := vendorprobe.Options{
		GeoIPDBPath:      s.cfg.GeoIP.DBPath,
		StripeSecretKey:  string(s.cfg.StripeSecretKey),
		TelegramBotToken: string(s.cfg.Notifier.TelegramBotToken),
		SMTPHost:         s.cfg.Notifier.SMTPHost,
		SMTPPort:         s.cfg.Notifier.SMTPPort,
	}
	reg := vendorprobe.RegistryFromOptions(opts)
	worker := vendorprobe.NewWorker(reg, vendorprobe.WorkerConfig{
		Interval: time.Duration(s.cfg.VendorTelemetryIntervalSec) * time.Second,
		Timeout:  time.Duration(s.cfg.VendorTelemetryTimeoutSec) * time.Second,
	}, vendorProbeMetrics{})
	s.startWorker(func() {
		worker.Start(s.ctx)
	})
}
