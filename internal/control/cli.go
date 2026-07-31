package control

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"espx/internal/config"
)

func ProbeHealth(args []string) bool {
	if len(args) > 2 && args[1] == "--health-probe" {
		resp, err := http.Get(args[2])
		if err != nil || resp.StatusCode != 200 {
			os.Exit(1)
		}
		os.Exit(0)
	}
	return false
}

func RunCLI(opts Options) {
	if ProbeHealth(os.Args) {
		return
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)
	slog.Warn("standalone binary deprecated; use cmd/control monolith")

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	RunLoaded(cfg, opts)
}

func RunLoaded(cfg *config.Config, opts Options) {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := Run(ctx, cfg, opts); err != nil && err != context.Canceled {
		slog.Error("control component stopped", "error", err)
		os.Exit(1)
	}
}

func OptionsAuthOnly() Options {
	return Options{Auth: true}
}

func OptionsManagementOnly() Options {
	return Options{Management: true}
}

func OptionsPaymentOnly() Options {
	return Options{Payment: true}
}

func OptionsBillingOnly() Options {
	return Options{Billing: true}
}

func OptionsNotifierOnly() Options {
	return Options{Notifier: true}
}
