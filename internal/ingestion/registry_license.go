package ingestion

import (
	"context"
	"crypto/ed25519"
	"log/slog"
	"time"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/licensing"
	"ad-event-processor/internal/metrics"
)

type fileLicenseSnapshot struct {
	state        licensing.LicenseState
	entitlements licensing.Entitlements
	featureSeed  uint32
	seedValid    bool
}

type RegistryLicenseConfig struct {
	Required bool
	Path     string
	PubKey   ed25519.PublicKey
	Interval time.Duration
}

func (r *Registry) ConfigureLicenseEnforcement(cfg RegistryLicenseConfig) {
	if r == nil || !cfg.Required {
		return
	}
	r.licenseEnforced.Store(true)
	if cfg.Path == "" {
		cfg.Path = config.LicensePathFromEnv()
	}
	if cfg.Interval <= 0 {
		cfg.Interval = 5 * time.Minute
	}
	r.fileLicense.Store(&fileLicenseSnapshot{state: licensing.StateExpired})
}

func (r *Registry) StartLicenseRecheck(ctx context.Context, cfg RegistryLicenseConfig) {
	if r == nil || !cfg.Required {
		return
	}
	if cfg.Path == "" {
		cfg.Path = config.LicensePathFromEnv()
	}
	if cfg.Interval <= 0 {
		cfg.Interval = 5 * time.Minute
	}

	r.ConfigureLicenseEnforcement(cfg)
	licensing.ConfigureSkewWatch(licensing.SkewWatchOptions{
		Enabled:   config.LicenseSkewWatchEnabled(),
		Interval:  config.LicenseSkewWatchInterval(),
		Threshold: config.LicenseSkewWatchThreshold(),
	})
	licensing.StartSkewWatch(ctx)
	r.recheckLicenseFile(ctx, cfg.Path, cfg.PubKey)

	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		ticker := time.NewTicker(cfg.Interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				r.recheckLicenseFile(ctx, cfg.Path, cfg.PubKey)
			}
		}
	}()
}

func (r *Registry) recheckLicenseFile(ctx context.Context, path string, pubKey ed25519.PublicKey) {
	now := time.Now()
	hostFP := licensing.HostFingerprint()
	licensing.SetSeedCouplingRequired(config.LicenseSeedCouplingEnabled())
	if licensing.EvaluateClockSkew() {
		metrics.LicenseClockSkewTotal.Inc()
		licensing.PublishFeatureSeed(0, false)
		slog.Warn("license clock skew detected")
		r.fileLicense.Store(&fileLicenseSnapshot{state: licensing.StateExpired, seedValid: false})
		return
	}
	if licensing.LicenseEpochInvalid() || licensing.GuardTripped() {
		licensing.PublishFeatureSeed(0, false)
		r.fileLicense.Store(&fileLicenseSnapshot{state: licensing.StateExpired, seedValid: false})
		return
	}
	verified, err := licensing.VerifyLicenseFile(path, pubKey, hostFP, now)
	if err == nil && verified.Claims != nil && r.pool != nil {
		if actErr := licensing.CheckHostActivation(ctx, r.pool, verified.Claims, hostFP); actErr != nil {
			err = actErr
		}
	}
	var seed uint32
	seedValid := !config.LicenseSeedCouplingEnabled()
	if err == nil && config.LicenseSeedCouplingEnabled() {
		if mck, mckErr := licensing.DeriveMCKFromLicenseFile(path, pubKey, hostFP); mckErr == nil {
			seed = licensing.FeatureSeedFromMCK(mck)
			seedValid = true
		}
	}
	licensing.PublishFeatureSeed(seed, seedValid)
	if err != nil {
		slog.Warn("deployment credential refresh failed", "error", err, "path", path)
		r.fileLicense.Store(&fileLicenseSnapshot{state: licensing.StateExpired, seedValid: false})
		return
	}
	r.fileLicense.Store(&fileLicenseSnapshot{
		state:        verified.State,
		entitlements: verified.Entitlements,
		featureSeed:  seed,
		seedValid:    seedValid,
	})
}

func (r *Registry) GetLicenseFeatureSeed() (uint32, bool) {
	if r == nil {
		return 0, false
	}
	if v, ok := r.fileLicense.Load().(*fileLicenseSnapshot); ok && v != nil {
		return v.featureSeed, v.seedValid
	}
	return 0, false
}
