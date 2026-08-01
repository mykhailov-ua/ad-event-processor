package ledger

import "log/slog"

type PaymentProvider interface {
	Name() string
	Configured() bool
}

type PlaceholderProvider struct {
	name string
	key  string
}

func NewPaymentProvider(providerName, providerKey string) PaymentProvider {
	if providerName == "" {
		providerName = "placeholder"
	}
	p := &PlaceholderProvider{name: providerName, key: providerKey}
	if providerName == "placeholder" {
		slog.Info("billing payment provider: placeholder mode", "key_set", providerKey != "")
	}
	return p
}

func (p *PlaceholderProvider) Name() string {
	if p == nil || p.name == "" {
		return "placeholder"
	}
	return p.name
}

func (p *PlaceholderProvider) Configured() bool {
	return false
}
