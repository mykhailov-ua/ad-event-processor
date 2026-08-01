package platformconfig

import (
	"fmt"
	"net"
	"strings"
)

func (c Config) Validate() error {
	if dom := strings.TrimSpace(c.TrackingDomain); dom != "" {
		dom = strings.TrimPrefix(strings.ToLower(dom), "https://")
		dom = strings.TrimPrefix(dom, "http://")
		dom = strings.TrimSuffix(dom, "/")
		if idx := strings.Index(dom, "/"); idx >= 0 {
			dom = dom[:idx]
		}
		if dom == "" || strings.Contains(dom, " ") {
			return fmt.Errorf("invalid tracking_domain")
		}
		c.TrackingDomain = dom
	}

	switch c.Profile {
	case ProfileSingleVPS, "compose_dev", "k8s_k3s":
	default:
		return fmt.Errorf("invalid profile: %s", c.Profile)
	}

	switch c.IngressSchema {
	case IngressESPXNative, IngressOpenRTB3:
	default:
		return fmt.Errorf("invalid ingress_schema: %s", c.IngressSchema)
	}

	if c.EdgeXDP && c.Profile == "compose_dev" {
		return fmt.Errorf("edge_xdp is not supported in compose_dev profile")
	}

	if c.Stripe.Enabled {
		if strings.TrimSpace(c.Stripe.SecretKey) == "" {
			return fmt.Errorf("stripe.secret_key is required when stripe is enabled")
		}
	}

	if cur := strings.TrimSpace(c.DefaultCurrency); cur != "" && len(cur) != 3 {
		return fmt.Errorf("default_currency must be a 3-letter ISO code")
	}

	return nil
}

func (p Patch) Apply(base Config) (Config, error) {
	out := base
	if p.TrackingDomain != nil {
		out.TrackingDomain = strings.TrimSpace(*p.TrackingDomain)
	}
	if p.DefaultCurrency != nil {
		out.DefaultCurrency = strings.ToUpper(strings.TrimSpace(*p.DefaultCurrency))
	}
	if p.Timezone != nil {
		out.Timezone = strings.TrimSpace(*p.Timezone)
	}
	if p.IngressSchema != nil {
		out.IngressSchema = strings.TrimSpace(*p.IngressSchema)
	}
	if p.TelemetryEnabled != nil {
		out.TelemetryEnabled = *p.TelemetryEnabled
	}
	if p.Profile != nil {
		out.Profile = strings.TrimSpace(*p.Profile)
	}
	if p.EdgeXDP != nil {
		out.EdgeXDP = *p.EdgeXDP
	}
	if p.NetworkInterface != nil {
		out.NetworkInterface = strings.TrimSpace(*p.NetworkInterface)
	}
	if p.Stripe != nil {
		if p.Stripe.Enabled != nil {
			out.Stripe.Enabled = *p.Stripe.Enabled
		}
		if p.Stripe.SecretKey != nil && strings.TrimSpace(*p.Stripe.SecretKey) != "" {
			out.Stripe.SecretKey = strings.TrimSpace(*p.Stripe.SecretKey)
		}
		if p.Stripe.WebhookSecret != nil && strings.TrimSpace(*p.Stripe.WebhookSecret) != "" {
			out.Stripe.WebhookSecret = strings.TrimSpace(*p.Stripe.WebhookSecret)
		}
		if p.Stripe.CheckoutSuccessURL != nil {
			out.Stripe.CheckoutSuccessURL = strings.TrimSpace(*p.Stripe.CheckoutSuccessURL)
		}
		if p.Stripe.CheckoutCancelURL != nil {
			out.Stripe.CheckoutCancelURL = strings.TrimSpace(*p.Stripe.CheckoutCancelURL)
		}
	}
	out = MergeDefaults(out)
	if err := out.Validate(); err != nil {
		return Config{}, err
	}
	return out, nil
}

func RestartRequiredFields(before, after Config) []string {
	var fields []string
	if before.IngressSchema != after.IngressSchema {
		fields = append(fields, "ingress_schema")
	}
	if before.TelemetryEnabled != after.TelemetryEnabled {
		fields = append(fields, "telemetry_enabled")
	}
	if before.EdgeXDP != after.EdgeXDP {
		fields = append(fields, "edge_xdp")
	}
	if before.Profile != after.Profile {
		fields = append(fields, "profile")
	}
	if before.NetworkInterface != after.NetworkInterface {
		fields = append(fields, "network_interface")
	}
	if stripeRestartRequired(before.Stripe, after.Stripe) {
		fields = append(fields, "stripe")
	}
	return fields
}

func stripeRestartRequired(before, after StripeConfig) bool {
	return before.Enabled != after.Enabled ||
		before.SecretKey != after.SecretKey ||
		before.WebhookSecret != after.WebhookSecret ||
		before.CheckoutSuccessURL != after.CheckoutSuccessURL ||
		before.CheckoutCancelURL != after.CheckoutCancelURL
}

func ClickURLTemplate(domain string) string {
	domain = strings.TrimSpace(domain)
	if domain == "" {
		return ""
	}
	return fmt.Sprintf("https://%s/click?campaign_id={campaign_id}&sub1={sub1}", domain)
}

func ResolveHost(domain string) string {
	domain = strings.TrimSpace(domain)
	domain = strings.TrimPrefix(strings.ToLower(domain), "https://")
	domain = strings.TrimPrefix(domain, "http://")
	if idx := strings.Index(domain, "/"); idx >= 0 {
		domain = domain[:idx]
	}
	if domain == "" {
		return ""
	}
	if net.ParseIP(domain) != nil {
		return domain
	}
	return domain
}
