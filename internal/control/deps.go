package control

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/licensing"
)

func RunFromCLI(ctx context.Context, cfg *config.Config) error {
	opts := OptionsFromConfig(cfg)
	slog.Info("starting control plane",
		"auth", opts.Auth,
		"management", opts.Management,
		"payment", opts.Payment,
		"billing", opts.Billing,
		"notifier", opts.Notifier,
		"margin_guard", opts.MarginGuard,
		"cost_sync", opts.CostSync,
	)

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

	if err := InitRuntimePolicy(); err != nil {
		slog.Error("failed to load control runtime policy", "error", err)
		if !config.LicenseAssetsUnsealed() {
			os.Exit(1)
		}
	}

	return Run(ctx, cfg, opts)
}

func NotifyContext() (context.Context, context.CancelFunc) {
	return signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
}

func RunCLI() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	ctx, cancel := NotifyContext()
	defer cancel()

	if err := RunFromCLI(ctx, cfg); err != nil && !errors.Is(err, context.Canceled) {
		slog.Error("control plane stopped", "error", err)
		os.Exit(1)
	}
}
