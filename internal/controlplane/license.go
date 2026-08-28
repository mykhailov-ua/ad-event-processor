package controlplane

import (
	"context"
	"crypto/ed25519"
	"errors"
	"log/slog"
	"net/http"
	"sync/atomic"
	"time"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/licensing"
	"ad-event-processor/internal/shardadmin"
	"ad-event-processor/pkg/httpresponse"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

var errLicenseWatcherUnavailable = errors.New("license watcher not configured")

const defaultLicenseRevokePoll = 30 * time.Second

var activeLicenseWatcher atomic.Pointer[licensing.LicenseWatcher]

func setLicenseWatcher(w *licensing.LicenseWatcher) {
	if w != nil {
		activeLicenseWatcher.Store(w)
	}
}

func startLicenseWatcher(ctx context.Context, pool *pgxpool.Pool, redisShards []redis.UniversalClient, pubKey ed25519.PublicKey, svc *Service) error {
	watcher := licensing.NewLicenseWatcher(pool, shardadmin.PickHealthyControlShard(redisShards), pubKey)
	watcher.SetControlRedisShards(redisShards)
	setLicenseWatcher(watcher)
	if err := watcher.Start(ctx); err != nil {
		return err
	}
	if config.LicenseRequiredFromEnv() {
		state, _ := watcher.GetState()
		if state == licensing.StateExpired || state == licensing.StateRevoked {
			slog.Warn("license required but ingest not allowed at startup", "state", state)
		}
	}
	if svc != nil {
		svc.StartBackgroundWorker(func() {
			<-ctx.Done()
		})
	}
	slog.Info("started license watcher", "mode", config.LicenseEnv("MODE"))
	return nil
}

func licenseIngestReady() bool {
	if !config.LicenseRequiredFromEnv() {
		return true
	}
	w := activeLicenseWatcher.Load()
	if w == nil {
		if licensing.SeedCouplingRequired() {
			return licensing.FeatureSeedValid()
		}
		return true
	}
	state, _ := w.GetState()
	if state == licensing.StateExpired || state == licensing.StateRevoked {
		return false
	}
	if licensing.SeedCouplingRequired() && !licensing.FeatureSeedValid() {
		return false
	}
	return true
}

func reloadLicense(ctx context.Context) error {
	w := activeLicenseWatcher.Load()
	if w == nil {
		return errLicenseWatcherUnavailable
	}
	return w.Reload(ctx)
}

func licenseWatcherState() (licensing.LicenseState, bool) {
	w := activeLicenseWatcher.Load()
	if w == nil {
		return licensing.StateExpired, false
	}
	state, _ := w.GetState()
	return state, true
}

func licenseWatcherDiagnostics() (licensing.LicenseDiagnostics, bool) {
	w := activeLicenseWatcher.Load()
	if w == nil {
		return licensing.LicenseDiagnostics{State: licensing.StateExpired}, false
	}
	state, claims := w.GetState()
	return licensing.BuildLicenseDiagnostics(claims, state, time.Now()), true
}

func licenseDeploymentLimits() (licensing.Limits, licensing.LicenseState, bool) {
	w := activeLicenseWatcher.Load()
	if w == nil {
		return licensing.Limits{}, licensing.StateExpired, false
	}
	state, claims := w.GetState()
	if claims == nil {
		return licensing.Limits{}, state, true
	}
	return claims.Limits, state, true
}

func activeActivationLicenseKey() string {
	w := activeLicenseWatcher.Load()
	if w == nil {
		return ""
	}
	_, claims := w.GetState()
	if claims == nil {
		return ""
	}
	return licensing.ActivationLicenseKey(claims)
}

func licenseFeatureAllowed(featureKey string) (allowed bool, planCode string) {
	w := activeLicenseWatcher.Load()
	if w == nil {
		return true, ""
	}
	state, claims := w.GetState()
	if claims == nil {
		return state == licensing.StateActive || state == licensing.StateGrace || state == licensing.StateOfflineWarn, ""
	}
	planCode = claims.Plan
	ent := licensing.Entitlements{Features: claims.Features}
	switch featureKey {
	case "openrtb":
		return licensing.OpenRTBAllowed(state, ent), planCode
	case "fraud_dispute_evidence":
		return licensing.FraudDisputeEvidenceAllowed(state, ent), planCode
	default:
		return true, planCode
	}
}

func writeLicenseFeatureRequired(w http.ResponseWriter, featureKey, planCode string) {
	httpresponse.JSON(w, http.StatusForbidden, LicenseFeatureRequiredBody{
		Error:       "feature_required",
		FeatureKey:  featureKey,
		PlanCode:    planCode,
		FeatureGate: featureKey,
	})
}

func requireLicenseFeature(w http.ResponseWriter, featureKey string) bool {
	allowed, planCode := licenseFeatureAllowed(featureKey)
	if allowed {
		return true
	}
	writeLicenseFeatureRequired(w, featureKey, planCode)
	return false
}
