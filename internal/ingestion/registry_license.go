package ingestion

import (
	"context"
	"crypto/ed25519"
	"time"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/licensing"
)

type fileLicenseSnapshot struct {
	state          licensing.LicenseState
	entitlements   licensing.Entitlements
	featureSeed    uint32
	mckFeatureBits uint8
	seedValid      bool
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
		base := 5 * time.Minute
		if d, err := time.ParseDuration(config.LicenseFileRecheckInterval()); err == nil && d > 0 {
			base = d
		}
		cfg.Interval = licensing.LicenseFileRecheckIntervalJittered(base, licensing.DeploymentIDFromLicensePath(cfg.Path))
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
		base := 5 * time.Minute
		if d, err := time.ParseDuration(config.LicenseFileRecheckInterval()); err == nil && d > 0 {
			base = d
		}
		cfg.Interval = licensing.LicenseFileRecheckIntervalJittered(base, licensing.DeploymentIDFromLicensePath(cfg.Path))
	}

	r.ConfigureLicenseEnforcement(cfg)

	recheckCfg := licensing.FileLicenseRecheckConfig{
		Path:     cfg.Path,
		PubKey:   cfg.PubKey,
		Interval: cfg.Interval,
	}
	if r.pool != nil {
		pool := r.pool
		recheckCfg.HostActivation = func(ctx context.Context, claims *licensing.LicenseClaims, hostFP string) error {
			return licensing.CheckHostActivation(ctx, pool, claims, hostFP)
		}
	}

	licensing.ConfigureSkewWatch(licensing.SkewWatchOptions{
		Enabled:   config.LicenseSkewWatchEnabled(),
		Interval:  config.LicenseSkewWatchInterval(),
		Threshold: config.LicenseSkewWatchThreshold(),
	})
	licensing.StartSkewWatch(ctx)
	licensing.SetSeedCouplingRequired(config.LicenseSeedCouplingEnabled())

	r.recheckLicenseFile(ctx, recheckCfg)

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
				r.recheckLicenseFile(ctx, recheckCfg)
			}
		}
	}()
}

func (r *Registry) recheckLicenseFile(ctx context.Context, cfg licensing.FileLicenseRecheckConfig) {
	snap, err := licensing.RecheckLicenseFile(ctx, cfg)
	if err != nil {
		r.fileLicense.Store(&fileLicenseSnapshot{state: licensing.StateExpired, seedValid: false})
		return
	}
	r.fileLicense.Store(&fileLicenseSnapshot{
		state:          snap.State,
		entitlements:   snap.Entitlements,
		featureSeed:    snap.FeatureSeed,
		mckFeatureBits: snap.MCKFeatureBits,
		seedValid:      snap.SeedValid,
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
