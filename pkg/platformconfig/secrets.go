package platformconfig

type PublicView struct {
	Config                  Config        `json:"config"`
	Secrets                 MaskedSecrets `json:"secrets"`
	RestartRequired         []string      `json:"restart_required,omitempty"`
	ClickURLTemplate        string        `json:"click_url_template,omitempty"`
	OpenRTBEndpointTemplate string        `json:"openrtb_endpoint_template,omitempty"`
	BootstrapComplete       bool          `json:"bootstrap_complete"`
}

type MaskedSecrets struct {
	StripeSecretKey     string `json:"stripe_secret_key,omitempty"`
	StripeWebhookSecret string `json:"stripe_webhook_secret,omitempty"`
}

func MaskSecret(value string) string {
	v := stringsTrim(value)
	if v == "" {
		return ""
	}
	if len(v) <= 4 {
		return "****"
	}
	return "****" + v[len(v)-4:]
}

func stringsTrim(s string) string {
	i := 0
	j := len(s)
	for i < j && (s[i] == ' ' || s[i] == '\t') {
		i++
	}
	for j > i && (s[j-1] == ' ' || s[j-1] == '\t') {
		j--
	}
	return s[i:j]
}

func Public(cfg Config, bootstrapComplete bool, pendingRestart []string) PublicView {
	view := PublicView{
		Config: RedactConfig(cfg),
		Secrets: MaskedSecrets{
			StripeSecretKey:     MaskSecret(cfg.Stripe.SecretKey),
			StripeWebhookSecret: MaskSecret(cfg.Stripe.WebhookSecret),
		},
		RestartRequired:         pendingRestart,
		ClickURLTemplate:        ClickURLTemplate(cfg.TrackingDomain),
		OpenRTBEndpointTemplate: OpenRTBEndpointTemplate(cfg.TrackingDomain),
		BootstrapComplete:       bootstrapComplete,
	}
	return view
}

func RedactConfig(cfg Config) Config {
	out := cfg
	out.Stripe.SecretKey = ""
	out.Stripe.WebhookSecret = ""
	return out
}
