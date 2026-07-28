package vendorprobe

// Options wires vendor probes from deployment configuration (cold path only).
type Options struct {
	GeoIPDBPath      string
	StripeSecretKey  string
	TelegramBotToken string
	SMTPHost         string
	SMTPPort         string
}

// RegistryFromOptions registers MaxMind, Stripe, Telegram, and SMTP probes.
func RegistryFromOptions(opts Options) *Registry {
	reg := NewRegistry()
	reg.Register(NewMaxMindProbe(opts.GeoIPDBPath))
	reg.Register(NewStripeProbe(opts.StripeSecretKey, nil))
	reg.Register(NewTelegramProbe(opts.TelegramBotToken, nil))
	reg.Register(NewSMTPProbe(opts.SMTPHost, opts.SMTPPort))
	return reg
}
