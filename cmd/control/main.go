package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/control"
	"ad-event-processor/internal/licensing"
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
		PtraceRequired: licensing.GuardCompiledIn() && config.LicenseGuardPtraceRequired(),
	})

	mode := config.LicenseMode()
	if mode == "dev" || mode == "development" {
		path := config.LicensePathFromEnv()
		if sig := licensing.RuntimeEntitlementSnapshot(path); sig != 0 {
			slog.Debug("runtime entitlement snapshot", "checksum", sig)
		}
		_ = licensing.DeploymentCredentialRefresh(path)
	}

	if config.LicenseRequiredFromEnv() {
		licensing.StartFileLicenseRecheck(ctx, licensing.FileLicenseRecheckConfig{
			Path: config.LicensePathFromEnv(),
		})
		slog.Info("license file recheck enabled", "path", config.LicensePathFromEnv())
	}

	if err := control.InitRuntimePolicy(); err != nil {
		slog.Error("failed to load control runtime policy", "error", err)
		if !config.LicenseAssetsUnsealed() {
			os.Exit(1)
		}
	}

	if err := control.Run(ctx, cfg, opts); err != nil && !errors.Is(err, context.Canceled) {
		slog.Error("control plane stopped", "error", err)
		os.Exit(1)
	}
}
