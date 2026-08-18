package licensing

import (
	"context"
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/bidshard/ad-event-processor/internal/config"
	"github.com/bidshard/ad-event-processor/internal/ledger/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

const registryFullSyncPayload = "*"

type LicenseWatcher struct {
	pool        *pgxpool.Pool
	rdb         redis.UniversalClient
	controlRdbs []redis.UniversalClient
	client      *LicenseClient
	mode        string
	path        string
	spoolDir    string
	serverURL   string
	licenseKey  string
	interval    time.Duration
	timeout     time.Duration
	policy      HeartbeatPolicy
	spool       *LicenseSpool

	mu               sync.RWMutex
	currentClaims    *LicenseClaims
	currentState     LicenseState
	lastVerifiedAt   time.Time
	lastRefreshError error
	offlineSince     time.Time
	pubKey           ed25519.PublicKey
}

func NewLicenseWatcher(pool *pgxpool.Pool, rdb redis.UniversalClient, pubKey ed25519.PublicKey) *LicenseWatcher {
	mode := config.LicenseEnv("MODE")
	if mode == "" {
		mode = "file"
	}
	path := config.LicenseEnv("PATH")
	if path == "" {
		path = "license.jwt"
	}
	spoolDir := config.LicenseEnv("SPOOL_DIR")
	if spoolDir == "" {
		spoolDir = filepath.Join(filepath.Dir(path), ".license-spool")
	}
	serverURL := config.LicenseEnv("SERVER")
	licenseKey := config.LicenseEnv("KEY")

	refreshStr := config.LicenseEnv("REFRESH_INTERVAL")
	interval := 24 * time.Hour
	if d, err := time.ParseDuration(refreshStr); err == nil {
		interval = d
	}

	timeoutStr := config.LicenseEnv("HEARTBEAT_TIMEOUT")
	timeout := 5 * time.Second
	if d, err := time.ParseDuration(timeoutStr); err == nil {
		timeout = d
	}
	if timeout > 30*time.Second {
		timeout = 30 * time.Second
	}

	client := NewLicenseClient(serverURL, licenseKey, timeout)

	return &LicenseWatcher{
		pool:         pool,
		rdb:          rdb,
		client:       client,
		mode:         mode,
		path:         path,
		spoolDir:     spoolDir,
		serverURL:    serverURL,
		licenseKey:   licenseKey,
		interval:     interval,
		timeout:      timeout,
		policy:       LoadHeartbeatPolicyFromEnv(),
		pubKey:       pubKey,
		currentState: StateExpired,
	}
}

func (w *LicenseWatcher) openSpool() error {
	if w.spool != nil {
		return nil
	}
	spool, err := OpenLicenseSpool(w.spoolDir)
	if err != nil {
		return fmt.Errorf("open license spool: %w", err)
	}
	w.spool = spool
	return nil
}

func (w *LicenseWatcher) closeSpool() {
	if w.spool != nil {
		_ = w.spool.Close()
		w.spool = nil
	}
}

func (w *LicenseWatcher) GetState() (LicenseState, *LicenseClaims) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.currentState, w.currentClaims
}

func (w *LicenseWatcher) Start(ctx context.Context) error {
	if err := w.openSpool(); err != nil {
		slog.Error("license spool open failed", "error", err)
	}

	if w.spool != nil {
		if token, err := w.spool.LatestToken(); err != nil {
			slog.Warn("license spool recovery failed", "error", err)
		} else if token != "" {
			if err := os.WriteFile(w.path, []byte(token), 0o600); err != nil {
				slog.Warn("failed to hydrate license file from spool", "error", err)
			} else {
				slog.Info("recovered license token from mmap spool")
			}
		}
	}

	if err := w.verifyAndReload(ctx); err != nil {
		slog.Error("Initial license verification failed", "error", err)
	}

	slog.Info("license watcher started",
		"mode", w.mode,
		"server", w.serverURL,
		"refresh_interval", w.interval.String(),
		"online", w.mode == "online" && w.licenseKey != "",
	)

	go func() {
		ticker := time.NewTicker(w.interval)
		defer ticker.Stop()
		defer w.closeSpool()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := w.verifyAndReload(ctx); err != nil {
					slog.Error("Scheduled license refresh failed", "error", err)
				}
			}
		}
	}()

	return nil
}

func (w *LicenseWatcher) Reload(ctx context.Context) error {
	return w.verifyAndReload(ctx)
}

func (w *LicenseWatcher) verifyAndReload(ctx context.Context) error {
	now := time.Now()
	var tokenStr string
	var err error
	heartbeatOffline := false

	if w.mode == "online" && w.licenseKey != "" {
		w.loadOfflineSince(ctx)
		tokenStr, err = w.performOnlineHeartbeat(ctx)
		if err != nil {
			slog.Warn("Online license heartbeat failed, falling back to cached file", "error", err)
			w.mu.Lock()
			w.lastRefreshError = err
			if w.offlineSince.IsZero() {
				w.offlineSince = now
			}
			heartbeatOffline = true
			w.mu.Unlock()
		} else {
			w.mu.Lock()
			w.offlineSince = time.Time{}
			w.mu.Unlock()
		}
	}

	if tokenStr == "" {
		tokenStr, err = w.readLocalFile()
		if err != nil {
			w.mu.Lock()
			w.lastRefreshError = err
			w.currentState = StateExpired
			w.mu.Unlock()
			SetLicenseMetrics(StateExpired, 0)
			return fmt.Errorf("failed to read local license file: %w", err)
		}
	}

	claims, err := VerifyJWTWithKey(tokenStr, w.pubKey)
	if err != nil {
		w.mu.Lock()
		w.lastRefreshError = err
		w.currentState = StateExpired
		w.mu.Unlock()
		SetLicenseMetrics(StateExpired, 0)
		return fmt.Errorf("license signature verification failed: %w", err)
	}

	if err := CheckHostActivation(ctx, w.pool, claims, HostFingerprint()); err != nil {
		w.mu.Lock()
		w.lastRefreshError = err
		w.currentState = StateExpired
		w.mu.Unlock()
		SetLicenseMetrics(StateExpired, 0)
		return fmt.Errorf("license bind verification failed: %w", err)
	}

	claims.Features = SanitizeFeaturesForSKU(claims.SKU, claims.Features)

	w.mu.Lock()
	offlineSince := w.offlineSince
	w.mu.Unlock()
	state := DetermineEffectiveState(claims, now, claims.Revoked, offlineSince, heartbeatOffline, w.policy)
	offlineDays := 0
	if heartbeatOffline {
		offlineDays = OfflineDays(offlineSince, now)
	}

	w.mu.Lock()
	w.currentClaims = claims
	w.currentState = state
	w.lastVerifiedAt = now
	if !heartbeatOffline {
		w.lastRefreshError = nil
	}
	w.mu.Unlock()

	SetLicenseMetrics(state, offlineDays)
	UpdateLogWatermark(claims)

	if w.pool == nil && w.rdb == nil {
		return nil
	}
	err = w.updateDatabaseAndRedis(ctx, tokenStr, claims, state, offlineSince, offlineDays)
	if err != nil {
		slog.Error("Failed to update license status in DB/Redis", "error", err)
		return err
	}

	return nil
}

func (w *LicenseWatcher) loadOfflineSince(ctx context.Context) {
	if w.rdb == nil {
		return
	}
	ts, err := w.rdb.HGet(ctx, "entitlement:deployment", "offline_since").Result()
	if err != nil || ts == "" {
		return
	}
	parsed, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return
	}
	w.mu.Lock()
	if w.offlineSince.IsZero() || parsed.Before(w.offlineSince) {
		w.offlineSince = parsed
	}
	w.mu.Unlock()
}

func (w *LicenseWatcher) performOnlineHeartbeat(ctx context.Context) (string, error) {
	cachedToken, err := w.readLocalFile()
	var deploymentID string
	var fingerprint string
	var uptime int64 = 300

	if err == nil {
		claims, err := DecodeUnverified(cachedToken)
		if err == nil {
			deploymentID = claims.DeploymentID
			fingerprint = claims.Bind.Fingerprint
		}
	}

	if deploymentID == "" {
		deploymentID = uuid.NewString()
	}

	var token string
	if cachedToken == "" {
		token, err = w.client.Activate(ctx, deploymentID, fingerprint)
		if err != nil {
			return "", err
		}
	} else {
		var notModified bool
		token, notModified, err = w.client.Heartbeat(ctx, deploymentID, fingerprint, uptime)
		if err != nil {
			return "", err
		}
		if notModified {
			return cachedToken, nil
		}
	}

	if err := w.persistLicenseToken(token); err != nil {
		slog.Error("Failed to cache license token", "error", err)
	}

	return token, nil
}

func (w *LicenseWatcher) persistLicenseToken(token string) error {
	if err := w.openSpool(); err != nil {
		return err
	}
	if w.spool != nil {
		if err := w.spool.AppendDurably(token); err != nil {
			return fmt.Errorf("spool append: %w", err)
		}
	}
	if err := WriteFileAtomic(w.path, []byte(token), 0o600); err != nil {
		return fmt.Errorf("write license file: %w", err)
	}
	return nil
}

func (w *LicenseWatcher) readLocalFile() (string, error) {
	if w.spool == nil {
		if err := w.openSpool(); err == nil && w.spool != nil {
			if token, spoolErr := w.spool.LatestToken(); spoolErr == nil && token != "" {
				return token, nil
			}
		}
	} else if token, err := w.spool.LatestToken(); err == nil && token != "" {
		return token, nil
	}
	data, err := os.ReadFile(w.path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (w *LicenseWatcher) updateDatabaseAndRedis(ctx context.Context, token string, claims *LicenseClaims, state LicenseState, offlineSince time.Time, offlineDays int) error {
	depID, err := uuid.Parse(claims.DeploymentID)
	if err != nil {
		return fmt.Errorf("invalid deployment id in claims: %w", err)
	}
	licID, err := uuid.Parse(claims.Subject)
	if err != nil {
		licID = uuid.Nil
	}

	entitlements := Entitlements{
		VolumeBand: ParseVolumeBand(string(claims.VolumeBand)),
		Limits:     claims.Limits,
		Features:   SanitizeFeaturesForSKU(claims.SKU, claims.Features).Normalized(),
	}
	entitlementsJSON, err := json.Marshal(entitlements)
	if err != nil {
		return err
	}

	queries := db.New(w.pool)
	var errStr pgtype.Text
	if w.lastRefreshError != nil {
		errStr = pgtype.Text{String: w.lastRefreshError.Error(), Valid: true}
	}

	_, err = queries.UpsertLicenseStatus(ctx, db.UpsertLicenseStatusParams{
		DeploymentID:     pgtype.UUID{Bytes: depID, Valid: true},
		LicenseID:        pgtype.UUID{Bytes: licID, Valid: true},
		PlanCode:         claims.Plan,
		ValidUntil:       pgtype.Timestamptz{Time: claims.ValidUntil, Valid: true},
		State:            string(state),
		EntitlementsJson: entitlementsJSON,
		LastVerifiedAt:   pgtype.Timestamptz{Time: w.lastVerifiedAt, Valid: true},
		LastRefreshError: errStr,
	})
	if err != nil {
		return fmt.Errorf("database update failed: %w", err)
	}

	redisKey := "entitlement:deployment"
	features := SanitizeFeaturesForSKU(claims.SKU, claims.Features).Normalized()
	fields := map[string]any{
		"state":                string(state),
		"plan":                 claims.Plan,
		"volume_band":          string(ParseVolumeBand(string(claims.VolumeBand))),
		"valid_until":          claims.ValidUntil.Format(time.RFC3339),
		"max_rps":              claims.Limits.MaxRPS,
		"max_requests_per_day": claims.Limits.MaxRequestsPerDay,
		"rtb_live":             boolToInt(features.RtbLive),
		"openrtb_engine":       boolToInt(features.OpenRTBEnabled()),
		"ivt_ml_detector":      boolToInt(features.IvtMLDetector),
		"ebpf_xdp_edge":        boolToInt(features.EbpfXDPEdge),
		"ml_fraud_boost":       boolToInt(features.MlFraudBoost),
		"multi_region":         boolToInt(features.MultiRegion),
		"slot_migration":       boolToInt(features.SlotMigration),
		"offline_days":         offlineDays,
		"banner_severity":      BannerSeverity(state),
	}
	if offlineSince.IsZero() {
		fields["offline_since"] = ""
	} else {
		fields["offline_since"] = offlineSince.UTC().Format(time.RFC3339)
	}

	if err := w.rdb.HMSet(ctx, redisKey, fields).Err(); err != nil {
		return fmt.Errorf("redis HMSet failed: %w", err)
	}

	w.publishCampaignUpdate(ctx)

	return nil
}

func (w *LicenseWatcher) SetControlRedisShards(rdbs []redis.UniversalClient) {
	w.controlRdbs = rdbs
}

func (w *LicenseWatcher) controlRedis() []redis.UniversalClient {
	if len(w.controlRdbs) > 0 {
		return w.controlRdbs
	}
	if w.rdb != nil {
		return []redis.UniversalClient{w.rdb}
	}
	return nil
}

func (w *LicenseWatcher) publishCampaignUpdate(ctx context.Context) {
	channel := "campaigns:update"
	for _, rdb := range w.controlRedis() {
		if rdb != nil {
			_ = rdb.Publish(ctx, channel, registryFullSyncPayload).Err()
		}
	}
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
