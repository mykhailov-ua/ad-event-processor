package platformconfig

import (
	"fmt"
	"strings"

	"github.com/bidshard/ad-event-processor/pkg/branding"
)

func RenderComposeEnv(cfg Config) []byte {
	lines := []string{
		branding.GeneratedConfigHeader(),
		"TRACKER_INGRESS_SCHEMA=" + cfg.IngressSchema,
		"AD_EVENT_PROCESSOR_TELEMETRY_ENABLED=" + boolString(cfg.TelemetryEnabled),
		"GOGC=300",
		"GOMEMLIMIT=700MiB",
		"PROCESSOR_GOGC=100",
	}
	if cfg.Stripe.Enabled {
		if v := strings.TrimSpace(cfg.Stripe.SecretKey); v != "" {
			lines = append(lines, "STRIPE_SECRET_KEY="+v)
		}
		if v := strings.TrimSpace(cfg.Stripe.WebhookSecret); v != "" {
			lines = append(lines, "STRIPE_WEBHOOK_SECRET="+v)
		}
		if v := strings.TrimSpace(cfg.Stripe.CheckoutSuccessURL); v != "" {
			lines = append(lines, "STRIPE_CHECKOUT_SUCCESS_URL="+v)
		}
		if v := strings.TrimSpace(cfg.Stripe.CheckoutCancelURL); v != "" {
			lines = append(lines, "STRIPE_CHECKOUT_CANCEL_URL="+v)
		}
	}
	if dom := strings.TrimSpace(cfg.TrackingDomain); dom != "" {
		lines = append(lines, "TRACKING_DOMAIN="+dom)
	}
	if cur := strings.TrimSpace(cfg.DefaultCurrency); cur != "" {
		lines = append(lines, "DEFAULT_CURRENCY="+strings.ToUpper(cur))
	}
	if tz := strings.TrimSpace(cfg.Timezone); tz != "" {
		lines = append(lines, "PLATFORM_TIMEZONE="+tz)
	}
	lines = append(lines, "REDIS_ADDRS="+RedisAddrsForProfile(cfg.Profile))
	lines = append(lines, "EDGE_EXPOSE_CLICK="+boolString(cfg.EdgeExposeClick))
	lines = append(lines, "EDGE_EXPOSE_OPENRTB="+boolString(cfg.EdgeExposeOpenRTB))
	return []byte(strings.Join(lines, "\n") + "\n")
}

func RenderInstallYAML(cfg Config) []byte {
	lines := []string{
		"profile: " + cfg.Profile,
		"ingress_schema: " + cfg.IngressSchema,
		"telemetry_enabled: " + boolString(cfg.TelemetryEnabled),
		"edge_xdp: " + boolString(cfg.EdgeXDP),
		"multi_region: false",
		"interface: " + cfg.NetworkInterface,
	}
	return []byte(strings.Join(lines, "\n") + "\n")
}

func boolString(v bool) string {
	if v {
		return "true"
	}
	return "false"
}

func FormatInstallRoot(path string) string {
	return strings.TrimRight(strings.TrimSpace(path), "/")
}

func ComposeEnvPath(installRoot string) string {
	root := FormatInstallRoot(installRoot)
	if root == "" {
		return "install.compose.env"
	}
	return fmt.Sprintf("%s/install.compose.env", root)
}
