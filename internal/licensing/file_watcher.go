package licensing

import (
	"context"
	"crypto/ed25519"
	"log/slog"
	"sync"
	"time"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/metrics"
)

// HostActivationFunc optionally enforces multi-host bind during file recheck (tracker registry).
type HostActivationFunc func(ctx context.Context, claims *LicenseClaims, hostFingerprint string) error

// FileLicenseRecheckConfig configures offline JWT file recheck on control/processor/tracker.
type FileLicenseRecheckConfig struct {
	Path           string
	PubKey         ed25519.PublicKey
	Interval       time.Duration
	HostActivation HostActivationFunc
}

// FileLicenseRecheckSnapshot is the registry-local license view after recheck.
type FileLicenseRecheckSnapshot struct {
	State          LicenseState
	Entitlements   Entitlements
	FeatureSeed    uint32
	MCKFeatureBits uint8
	SeedValid      bool
}

var fileLicenseRecheckWG sync.WaitGroup

// StartFileLicenseRecheck runs periodic license file verification and publishes feature seed atomics.
func StartFileLicenseRecheck(ctx context.Context, cfg FileLicenseRecheckConfig) {
	if cfg.Path == "" {
		cfg.Path = config.LicensePathFromEnv()
	}
	if cfg.Interval <= 0 {
		base := fileLicenseRecheckInterval()
		cfg.Interval = LicenseFileRecheckIntervalJittered(base, DeploymentIDFromLicensePath(cfg.Path))
	}

	ConfigureSkewWatch(SkewWatchOptions{
		Enabled:   config.LicenseSkewWatchEnabled(),
		Interval:  config.LicenseSkewWatchInterval(),
		Threshold: config.LicenseSkewWatchThreshold(),
	})
	StartSkewWatch(ctx)
	SetSeedCouplingRequired(config.LicenseSeedCouplingEnabled())

	run := func() {
		if _, err := RecheckLicenseFile(ctx, cfg); err != nil {
			slog.Debug("license file recheck", "error", err, "path", cfg.Path)
		}
	}
	run()

	fileLicenseRecheckWG.Add(1)
	go func() {
		defer fileLicenseRecheckWG.Done()
		ticker := time.NewTicker(cfg.Interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				run()
			}
		}
	}()
}

func fileLicenseRecheckInterval() time.Duration {
	if d, err := time.ParseDuration(config.LicenseFileRecheckInterval()); err == nil && d > 0 {
		return d
	}
	return 5 * time.Minute
}

// RecheckLicenseFile verifies the on-disk JWT, stretches MCK on success, and publishes feature seed.
func RecheckLicenseFile(ctx context.Context, cfg FileLicenseRecheckConfig) (FileLicenseRecheckSnapshot, error) {
	var snap FileLicenseRecheckSnapshot
	snap.State = StateExpired

	if cfg.Path == "" {
		cfg.Path = config.LicensePathFromEnv()
	}

	SetSeedCouplingRequired(config.LicenseSeedCouplingEnabled())

	if EvaluateClockSkew() {
		metrics.LicenseClockSkewTotal.Inc()
		PublishFeatureSeed(0, false)
		slog.Warn("license clock skew detected")
		return snap, nil
	}
	if LicenseEpochInvalid() || GuardTripped() {
		PublishFeatureSeed(0, false)
		return snap, nil
	}

	now := time.Now()
	hostFP := HostFingerprint()
	verified, err := VerifyLicenseFile(cfg.Path, cfg.PubKey, hostFP, now)
	if err == nil && verified.Claims != nil && cfg.HostActivation != nil {
		if actErr := cfg.HostActivation(ctx, verified.Claims, hostFP); actErr != nil {
			err = actErr
		}
	}
	if err == nil && config.LicenseSeedCouplingEnabled() {
		if macErr := verifyOrBootstrapLicenseMAC(cfg.Path, cfg.PubKey, hostFP); macErr != nil {
			err = macErr
		}
	}

	seed, mckBits, seedValid := featureSeedFromRecheck(cfg.Path, cfg.PubKey, hostFP, err)
	PublishFeatureSeed(seed, seedValid)
	if seedValid {
		PublishMCKFeatureBits(mckBits)
	}

	if err != nil {
		slog.Warn("deployment credential refresh failed", "error", err, "path", cfg.Path)
		return snap, err
	}

	snap = FileLicenseRecheckSnapshot{
		State:          verified.State,
		Entitlements:   verified.Entitlements,
		FeatureSeed:    seed,
		MCKFeatureBits: mckBits,
		SeedValid:      seedValid,
	}
	return snap, nil
}

func featureSeedFromRecheck(path string, pubKey ed25519.PublicKey, hostFP string, verifyErr error) (uint32, uint8, bool) {
	if !config.LicenseSeedCouplingEnabled() {
		return 0, 0, true
	}
	if verifyErr != nil {
		return 0, 0, false
	}
	mckWork, err := DeriveMCKWorkForRecheckFromLicenseFile(path, pubKey, hostFP)
	if err != nil {
		return 0, 0, false
	}
	return FeatureSeedFromMCK(mckWork), MCKFeatureBitsFromWork(mckWork), true
}

// WaitFileLicenseRecheckForTest blocks until background recheck goroutines exit.
func WaitFileLicenseRecheckForTest() {
	fileLicenseRecheckWG.Wait()
}
