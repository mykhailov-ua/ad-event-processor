package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/bidshard/ad-event-processor/internal/config"
	"github.com/bidshard/ad-event-processor/internal/control"
	"github.com/bidshard/ad-event-processor/internal/licensing"
)

func main() {
	if control.ProbeHealth(os.Args) {
		return
	}
	if licensing.MaybeRunGuardWatchdogCLI(os.Args) {
		return
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	opts := control.OptionsFromConfig(cfg)
	slog.Info("starting control plane",
		"auth", opts.Auth,
		"management", opts.Management,
		"payment", opts.Payment,
		"billing", opts.Billing,
		"notifier", opts.Notifier,
		"margin_guard", opts.MarginGuard,
		"cost_sync", opts.CostSync,
	)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	licensing.StartLicenseGuard(ctx, licensing.GuardConfig{
		Enabled:        licensing.GuardCompiledIn() && config.LicenseGuardEnvEnabled(),
		PtraceWatchdog: licensing.GuardCompiledIn() && config.LicenseGuardPtraceWatchdogEnabled(),
	})

	if err := control.Run(ctx, cfg, opts); err != nil && !errors.Is(err, context.Canceled) {
		slog.Error("control plane stopped", "error", err)
		os.Exit(1)
	}
}
