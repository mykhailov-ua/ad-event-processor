package ingestion

import (
	"context"
	"crypto/ed25519"
	"log/slog"
	"time"

	"espx/internal/config"
	"espx/internal/licensing"
)

type fileLicenseSnapshot struct {
	state        licensing.LicenseState
	entitlements licensing.Entitlements
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
		cfg.Path = config.LicenseEnv("PATH")
	}
	if cfg.Path == "" {
		cfg.Path = "license.jwt"
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
		cfg.Path = config.LicenseEnv("PATH")
	}
	if cfg.Path == "" {
		cfg.Path = "license.jwt"
	}
	if cfg.Interval <= 0 {
		cfg.Interval = 5 * time.Minute
	}

	r.ConfigureLicenseEnforcement(cfg)
	r.recheckLicenseFile(cfg.Path, cfg.PubKey)

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
				r.recheckLicenseFile(cfg.Path, cfg.PubKey)
			}
		}
	}()
}

func (r *Registry) recheckLicenseFile(path string, pubKey ed25519.PublicKey) {
	now := time.Now()
	hostFP := licensing.HostFingerprint()
	verified, err := licensing.VerifyLicenseFile(path, pubKey, hostFP, now)
	if err != nil {
		slog.Warn("license file verification failed", "error", err, "path", path)
		r.fileLicense.Store(&fileLicenseSnapshot{state: licensing.StateExpired})
		return
	}
	r.fileLicense.Store(&fileLicenseSnapshot{
		state:        verified.State,
		entitlements: verified.Entitlements,
	})
}
