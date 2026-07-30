package control

import (
	"espx/internal/config"
)

type Options struct {
	Auth        bool
	Management  bool
	Payment     bool
	Billing     bool
	Notifier    bool
	MarginGuard bool
	CostSync    bool
}

func OptionsFromConfig(cfg *config.Config) Options {
	if cfg == nil {
		return Options{Auth: true, Management: true}
	}
	return Options{
		Auth:        cfg.Control.EnableAuth,
		Management:  cfg.Control.EnableManagement,
		Payment:     cfg.Control.EnablePayment,
		Billing:     cfg.Control.EnableBilling,
		Notifier:    cfg.Control.EnableNotifier,
		MarginGuard: cfg.Control.EnableMarginGuard,
		CostSync:    cfg.Control.EnableCostSync,
	}
}
