package filter

import (
	"context"
	"crypto/ed25519"
	"encoding/json"
	"hash/fnv"
	"log/slog"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/licensing"
	"ad-event-processor/internal/metrics"

	"github.com/google/uuid"
	redis "github.com/redis/go-redis/v9"
)

func (r *Registry) BootstrapFromReplica() (int, error) {
	if r == nil {
		return 0, nil
	}
	if len(r.campaignMapSnapshot().byID) > 0 {
		return len(r.campaignMapSnapshot().byID), nil
	}
	loaded, err := r.loadReplica()
	if err != nil {
		return 0, err
	}
	r.storeCampaignSnapshot(loaded)
	n := len(loaded.byID)
	if n > 0 {
		slog.Info("campaign registry bootstrapped from local replica", "campaigns", n)
	}
	return n, nil
}

func (r *Registry) StartWatchShards(ctx context.Context, redisShards []redis.UniversalClient, channel string) {
	for shardIdx, redisClient := range redisShards {
		if redisClient == nil {
			continue
		}
		r.wg.Add(1)
		staleDriver := true
		go r.runShardPubSubWatch(ctx, redisClient, channel, shardIdx, staleDriver)
	}
}

func (r *Registry) runShardPubSubWatch(ctx context.Context, redisClient redis.UniversalClient, channel string, shardIdx int, staleDriver bool) {
	defer r.wg.Done()
	backoff := time.Second
	for {
		if ctx.Err() != nil {
			return
		}
		err := r.watchPubSubOnce(ctx, redisClient, channel, staleDriver)
		if ctx.Err() != nil {
			return
		}
		slog.Warn("registry pubsub disconnected, reconnecting", "error", err, "shard", shardIdx, "backoff", backoff)
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
		if backoff < 30*time.Second {
			backoff *= 2
		}
	}
}

func (r *Registry) StartEpochPoll(ctx context.Context, redisShards []redis.UniversalClient, interval time.Duration) {
	if interval <= 0 || len(redisShards) == 0 {
		return
	}
	epochs := make([]atomic.Uint64, len(redisShards))
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				r.pollRegistryEpochs(ctx, redisShards, epochs)
			}
		}
	}()
}

func (r *Registry) pollRegistryEpochs(ctx context.Context, redisShards []redis.UniversalClient, epochs []atomic.Uint64) {
	var maxEpoch uint64
	for shardIdx, redisClient := range redisShards {
		if redisClient == nil {
			continue
		}
		val, err := redisClient.Get(ctx, CampaignEpochKey).Uint64()
		if err != nil {
			continue
		}
		if val > maxEpoch {
			maxEpoch = val
		}
		local := epochs[shardIdx].Load()
		if val > local {
			epochs[shardIdx].Store(val)
			if _, err := r.ReloadFullSnapshot(ctx); err != nil {
				slog.Error("campaign registry epoch poll reload failed", "shard", shardIdx, "error", err)
				continue
			}
			r.MarkPubSubOK()
			slog.Debug("campaign registry epoch poll reload", "shard", strconv.Itoa(shardIdx), "epoch", val)
		}
	}
	if maxEpoch > 0 {
		metrics.RegistryEpoch.Set(float64(maxEpoch))
	}
}

func (r *Registry) StartWatch(ctx context.Context, redisClient redis.UniversalClient, channel string) {
	if redisClient == nil {
		return
	}
	r.StartWatchShards(ctx, []redis.UniversalClient{redisClient}, channel)
}

const defaultRegistryStaleTTL = 30 * time.Second

func (r *Registry) ConfigureStaleMode(ttl time.Duration) {
	if r == nil {
		return
	}
	if ttl <= 0 {
		ttl = defaultRegistryStaleTTL
	}
	atomic.StoreInt64(&r.staleTTLNano, int64(ttl))
	now := time.Now().UnixNano()
	atomic.StoreInt64(&r.lastPubSubOKUnix, now)
	r.refreshStaleMode(now)
}

func (r *Registry) MarkPubSubOK() {
	if r == nil {
		return
	}
	now := time.Now().UnixNano()
	atomic.StoreInt64(&r.lastPubSubOKUnix, now)
	r.refreshStaleMode(now)
}

func (r *Registry) IsStaleMode() bool {
	if r == nil {
		return false
	}
	ttl := atomic.LoadInt64(&r.staleTTLNano)
	if ttl <= 0 {
		return false
	}
	now := time.Now().UnixNano()
	r.refreshStaleMode(now)
	return atomic.LoadInt32(&r.staleMode) == 1
}

func (r *Registry) refreshStaleMode(nowUnixNano int64) {
	ttl := atomic.LoadInt64(&r.staleTTLNano)
	if ttl <= 0 {
		if atomic.SwapInt32(&r.staleMode, 0) == 1 {
			metrics.RegistryStaleMode.Set(0)
			metrics.Shard0PubSubUnreachable.Set(0)
		}
		return
	}
	last := atomic.LoadInt64(&r.lastPubSubOKUnix)
	stale := last > 0 && nowUnixNano-last > ttl
	want := int32(0)
	if stale {
		want = 1
	}
	prev := atomic.SwapInt32(&r.staleMode, want)
	if prev != want {
		metrics.RegistryStaleMode.Set(float64(want))
		metrics.Shard0PubSubUnreachable.Set(float64(want))
	}
}

func (r *Registry) ReloadFullSnapshot(ctx context.Context) (int, error) {
	count, err := r.Sync(ctx)
	if err != nil {
		return count, err
	}

	r.mu.Lock()
	w := r.budgetWarmer
	r.mu.Unlock()

	if w != nil {
		if warmed, warmErr := w.WarmFromRegistry(ctx, r); warmErr != nil {
			slog.Warn("registry full sync: budget warm failed", "error", warmErr)
		} else if warmed > 0 {
			slog.Debug("registry full sync: budget keys warmed", "keys", warmed)
		}
	}

	slog.Info("campaign registry full sync", "campaigns", count)
	return count, nil
}

const registryWorkerCacheMax = 128

type registryWorkerCacheEntry struct {
	id   uuid.UUID
	gen  uint64
	camp *domain.Campaign
}

type registryWorkerCacheSlot struct {
	ptr atomic.Pointer[registryWorkerCacheEntry]
}

func (r *Registry) storeCampaignSnapshot(s *campaignMapSnapshot) {
	if r == nil {
		return
	}
	r.data.Store(s)
	r.snapGen.Add(1)
}

func (r *Registry) GetCampaignWorker(worker int, id uuid.UUID) (*domain.Campaign, bool) {
	if r == nil {
		return nil, false
	}
	gen := r.snapGen.Load()
	if worker >= 0 && worker < registryWorkerCacheMax {
		if ent := r.workerCache[worker].ptr.Load(); ent != nil && ent.id == id && ent.gen == gen && ent.camp != nil {
			return ent.camp, true
		}
	}
	info, ok := r.campaignMapSnapshot().byID[id]
	if !ok || info.campaign == nil {
		return nil, false
	}
	if worker >= 0 && worker < registryWorkerCacheMax {
		r.workerCache[worker].ptr.Store(&registryWorkerCacheEntry{
			id:   id,
			gen:  gen,
			camp: info.campaign,
		})
	}
	return info.campaign, true
}

func getCampaignFromEvent(registry domain.CampaignRegistry, evt *domain.Event) (*domain.Campaign, bool) {
	if registry == nil || evt == nil {
		return nil, false
	}
	if evt.FilterCampResolved {
		if evt.FilterCamp == nil {
			return nil, false
		}
		return evt.FilterCamp, true
	}
	var camp *domain.Campaign
	var ok bool
	if reg, isReg := registry.(*Registry); isReg {
		if w := int(evt.FilterWorkerIdx); w >= 0 {
			camp, ok = reg.GetCampaignWorker(w, evt.CampaignID)
		} else {
			camp, ok = reg.GetCampaign(evt.CampaignID)
		}
	} else {
		camp, ok = registry.GetCampaign(evt.CampaignID)
	}
	evt.FilterCampResolved = true
	if ok {
		evt.FilterCamp = camp
	}
	return camp, ok
}

func GetCampaignFromEvent(registry domain.CampaignRegistry, evt *domain.Event) (*domain.Campaign, bool) {
	return getCampaignFromEvent(registry, evt)
}

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

type mockRegistry struct{}

func (m *mockRegistry) Exists(id uuid.UUID) bool { return true }
func (m *mockRegistry) Add(id, customerID uuid.UUID, brandID *uuid.UUID, brandFcapKey string, pacingMode domain.PacingMode, dailyBudget int64, timezone string, freqLimit, freqWindow int32, targetCountries []string) {
}
func (m *mockRegistry) GetCustomerID(id uuid.UUID) (uuid.UUID, bool) { return uuid.Nil, true }

var (
	staticCampaignMu sync.RWMutex
	staticCampaign   = &domain.Campaign{CustomerID: uuid.Nil, Location: time.UTC}
	cachedMockCamp   atomic.Pointer[domain.Campaign]
)

func ResetStaticCampaignBaseline() {
	resetStaticCampaignBaseline()
}

func resetStaticCampaignBaseline() {
	lockStaticCampaign(func(c *domain.Campaign) {
		*c = domain.Campaign{CustomerID: uuid.Nil, Location: time.UTC}
	})
	cachedMockCamp.Store(nil)
}

func LockStaticCampaign(mut func(c *domain.Campaign)) {
	lockStaticCampaign(mut)
}

func SetStaticCampaignForTest(camp *domain.Campaign) {
	staticCampaignMu.Lock()
	defer staticCampaignMu.Unlock()
	if camp == nil {
		staticCampaign = &domain.Campaign{CustomerID: uuid.Nil, Location: time.UTC}
	} else {
		staticCampaign = camp
	}
	cachedMockCamp.Store(nil)
}

func WithStaticCampaign(fn func(camp **domain.Campaign)) {
	staticCampaignMu.Lock()
	defer staticCampaignMu.Unlock()
	fn(&staticCampaign)
}

func CachedMockCamp() *atomic.Pointer[domain.Campaign] {
	return &cachedMockCamp
}

func lockStaticCampaign(mut func(c *domain.Campaign)) {
	staticCampaignMu.Lock()
	defer staticCampaignMu.Unlock()
	mut(staticCampaign)
}

func ConfigureMockRegistryCampaign(mut func(c *domain.Campaign)) {
	configureMockRegistryCampaign(mut)
}

func configureMockRegistryCampaign(mut func(c *domain.Campaign)) {
	lockStaticCampaign(func(c *domain.Campaign) {
		*c = domain.Campaign{CustomerID: uuid.Nil, Location: time.UTC}
		mut(c)
	})
	cachedMockCamp.Store(nil)
}

func EnrichMockCampaign(cp *domain.Campaign) {
	enrichMockCampaign(cp)
}

func enrichMockCampaign(cp *domain.Campaign) {
	if cp.Location == nil {
		cp.Location = time.UTC
	}
	if cp.IDStr == "" {
		cp.IDStr = cp.ID.String()
	}
	if cp.IDStrAny == nil {
		cp.IDStrAny = cp.IDStr
	}
	if cp.CustomerIDStr == "" {
		cp.CustomerIDStr = cp.CustomerID.String()
	}
	if cp.CustomerIDStrAny == nil {
		cp.CustomerIDStrAny = cp.CustomerIDStr
	}
	if cp.BudgetCampaignKey == "" {
		cp.BudgetCampaignKey = "budget:campaign:" + cp.IDStr
	}
	if cp.CampaignSyncKey == "" {
		cp.CampaignSyncKey = "budget:sync:campaign:" + cp.IDStr
	}
	if cp.CustomerSyncKey == "" {
		cp.CustomerSyncKey = "budget:sync:customer:" + cp.CustomerIDStr
	}
	if cp.FcapKeyPrefix == "" {
		if cp.BrandFcapKey != "" {
			cp.FcapKeyPrefix = cp.BrandFcapKey + ":u:"
		} else {
			cp.FcapKeyPrefix = "fcap:c:" + cp.IDStr + ":u:"
		}
	}
	if cp.DailySpendKeyPrefix == "" {
		cp.DailySpendKeyPrefix = "budget:daily_spent:campaign:" + cp.IDStr + ":"
	}
	if cp.DailyBudgetMicroAny == nil && cp.DailyBudgetMicro != 0 {
		cp.DailyBudgetMicroAny = cp.DailyBudgetMicro
	}
}

func (m *mockRegistry) GetCampaign(id uuid.UUID) (*domain.Campaign, bool) {
	if got := cachedMockCamp.Load(); got != nil && got.ID == id {
		if got.BudgetCampaignKey == "" {
			cp := *got
			enrichMockCampaign(&cp)
			cachedMockCamp.Store(&cp)
		}
		return cachedMockCamp.Load(), true
	}

	staticCampaignMu.RLock()
	defer staticCampaignMu.RUnlock()

	cp := *staticCampaign
	cp.ID = id
	enrichMockCampaign(&cp)

	cachedMockCamp.Store(&cp)
	return cachedMockCamp.Load(), true
}
func (m *mockRegistry) Sync(ctx context.Context) (int, error)                 { return 0, nil }
func (m *mockRegistry) StartSync(ctx context.Context, interval time.Duration) {}
func (m *mockRegistry) Wait(ctx context.Context) error                        { return nil }

func AssignCohortVariant(salt, subjectID string, variants []domain.CohortVariant) (variantID string, flags map[string]string) {
	if len(variants) == 0 {
		return "", nil
	}
	var total uint32
	for _, v := range variants {
		total += v.Weight
	}
	if total == 0 {
		v := variants[0]
		return v.ID, cloneFlags(v.Flags)
	}

	h := fnv.New32a()
	_, _ = h.Write([]byte(salt))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write([]byte(subjectID))
	bucket := h.Sum32() % total

	var cursor uint32
	for _, v := range variants {
		cursor += v.Weight
		if bucket < cursor {
			return v.ID, cloneFlags(v.Flags)
		}
	}
	last := variants[len(variants)-1]
	return last.ID, cloneFlags(last.Flags)
}

func cloneFlags(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

type cohortVariantDTO struct {
	ID     string            `json:"id"`
	Weight uint32            `json:"weight"`
	Flags  map[string]string `json:"flags,omitempty"`
}

type cohortRegistrySnapshot struct {
	byID map[uuid.UUID]domain.ExperimentCohort
}

func (r *Registry) SyncCohorts(ctx context.Context) error {
	if r == nil || r.repo == nil {
		return nil
	}
	listFn, ok := r.repo.(interface {
		ListActiveExperimentCohorts(context.Context) ([]db.ExperimentCohort, error)
	})
	if !ok {
		return nil
	}
	rows, err := listFn.ListActiveExperimentCohorts(ctx)
	if err != nil {
		return err
	}
	byID := make(map[uuid.UUID]domain.ExperimentCohort, len(rows))
	for _, row := range rows {
		id := uuid.UUID(row.ID.Bytes)
		var variants []cohortVariantDTO
		if err := json.Unmarshal(row.Variants, &variants); err != nil {
			slog.Warn("skip invalid experiment cohort row", "id", id, "error", err)
			continue
		}
		cohort := domain.ExperimentCohort{
			ID:       id,
			Name:     row.Name,
			Salt:     row.Salt,
			Variants: make([]domain.CohortVariant, 0, len(variants)),
		}
		for _, v := range variants {
			if v.ID == "" || v.Weight == 0 {
				continue
			}
			cohort.Variants = append(cohort.Variants, domain.CohortVariant{
				ID:     v.ID,
				Weight: v.Weight,
				Flags:  v.Flags,
			})
		}
		if len(cohort.Variants) == 0 {
			slog.Warn("skip invalid experiment cohort row", "id", id, "error", "no valid variants")
			continue
		}
		byID[cohort.ID] = cohort
	}
	r.cohorts.Store(&cohortRegistrySnapshot{byID: byID})
	return nil
}

func (r *Registry) cohortSnapshot() *cohortRegistrySnapshot {
	if r == nil {
		return &cohortRegistrySnapshot{}
	}
	v, ok := r.cohorts.Load().(*cohortRegistrySnapshot)
	if !ok || v == nil {
		return &cohortRegistrySnapshot{}
	}
	return v
}

func (r *Registry) AssignExperiment(experimentID uuid.UUID, subjectID string) (variantID string, flags map[string]string, ok bool) {
	if r == nil || subjectID == "" {
		return "", nil, false
	}
	cohort, found := r.cohortSnapshot().byID[experimentID]
	if !found {
		return "", nil, false
	}
	variantID, flags = AssignCohortVariant(cohort.Salt, subjectID, cohort.Variants)
	return variantID, flags, variantID != ""
}

func (r *Registry) ExperimentCount() int {
	return len(r.cohortSnapshot().byID)
}
