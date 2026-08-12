package ingestion

import (
	"context"
	"log/slog"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/bidshard/ad-event-processor/internal/config"
	"github.com/bidshard/ad-event-processor/internal/domain"
	"github.com/bidshard/ad-event-processor/internal/rtb"
	"github.com/bidshard/ad-event-processor/pkg/money"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type DynamicConfig struct {
	Version             int64  `json:"version"`
	RateLimitPerMin     int    `json:"rate_limit_per_min"`
	RateLimitWindow     int    `json:"rate_limit_window_ms"`
	ClickAmount         int64  `json:"click_amount"`
	ImpressionAmount    int64  `json:"impression_amount"`
	EmergencyBreaker    bool   `json:"emergency_breaker"`
	FraudRLSuspectPct   int    `json:"fraud_rl_suspect_pct"`
	FraudRLIVTPct       int    `json:"fraud_rl_ivt_pct"`
	FraudRLBlockPct     int    `json:"fraud_rl_block_pct"`
	FraudRLRetrySuspect int    `json:"fraud_rl_retry_suspect_sec"`
	FraudRLRetryIVT     int    `json:"fraud_rl_retry_ivt_sec"`
	FraudRLRetryBlock   int    `json:"fraud_rl_retry_block_sec"`
	ASNCDNWhitelist     string `json:"asn_cdn_whitelist"`
	ASNMobileWhitelist  string `json:"asn_mobile_whitelist"`
	TLSHashBlocklist    string `json:"tls_hash_blocklist"`
	RtbBudgetAuthority  string `json:"rtb_budget_authority"`
	RtbMode             string `json:"rtb_mode"`
}

type SettingsChangeListener func(*DynamicConfig)

type FraudScoreBoostSnapshot struct {
	Boosts map[uuid.UUID]uint8
}

type VPPRatioSnapshot struct {
	Ratios map[uuid.UUID]float32
}

type SettingsWatcher struct {
	rdbs             []redis.UniversalClient
	currentVersion   int64
	snapshot         atomic.Value
	fraudScoreBoosts atomic.Value
	vppRatios        atomic.Value
	fcapSnap         atomic.Pointer[rtb.FcapSnapshot]
	onChange         []SettingsChangeListener
	pgSync           func(context.Context) (map[string]string, int64, error)
	staleCheck       func() bool
}

func NewSettingsWatcher(rdbs []redis.UniversalClient, initial *config.Config) *SettingsWatcher {
	sw := &SettingsWatcher{
		rdbs: rdbs,
	}

	sw.snapshot.Store(&DynamicConfig{
		Version:          0,
		RateLimitPerMin:  initial.RateLimitPerMin,
		RateLimitWindow:  initial.RateLimitWindowMs,
		ClickAmount:      initial.ClickAmount,
		ImpressionAmount: initial.ImpressionAmount,
		EmergencyBreaker: false,
	})

	sw.fraudScoreBoosts.Store(&FraudScoreBoostSnapshot{
		Boosts: make(map[uuid.UUID]uint8),
	})
	sw.vppRatios.Store(&VPPRatioSnapshot{Ratios: make(map[uuid.UUID]float32)})
	sw.fcapSnap.Store(emptyFcapSnapshot)

	return sw
}

func (sw *SettingsWatcher) SetPGFallback(syncFn func(context.Context) (map[string]string, int64, error), staleCheck func() bool) {
	if sw == nil {
		return
	}
	sw.pgSync = syncFn
	sw.staleCheck = staleCheck
}

func (sw *SettingsWatcher) AddChangeListener(fn SettingsChangeListener) {
	if fn == nil {
		return
	}
	sw.onChange = append(sw.onChange, fn)
}

func (sw *SettingsWatcher) Get() *DynamicConfig {
	return sw.snapshot.Load().(*DynamicConfig)
}

func (sw *SettingsWatcher) GetFraudScoreBoosts() *FraudScoreBoostSnapshot {
	v := sw.fraudScoreBoosts.Load()
	if v == nil {
		return &FraudScoreBoostSnapshot{Boosts: make(map[uuid.UUID]uint8)}
	}
	return v.(*FraudScoreBoostSnapshot)
}

func (sw *SettingsWatcher) GetVPPRatio(campaignID uuid.UUID) float32 {
	v := sw.vppRatios.Load()
	if v == nil {
		return 1.0
	}
	snap := v.(*VPPRatioSnapshot)
	if snap == nil || snap.Ratios == nil {
		return 1.0
	}
	ratio, ok := snap.Ratios[campaignID]
	if !ok {
		return 1.0
	}
	return ratio
}

func (sw *SettingsWatcher) GetFcapRtbSnapshot() *rtb.FcapSnapshot {
	if sw == nil {
		return emptyFcapSnapshot
	}
	snap := sw.fcapSnap.Load()
	if snap == nil {
		return emptyFcapSnapshot
	}
	return snap
}

func (sw *SettingsWatcher) Start(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sw.sync(ctx)
			sw.syncFraudScoreBoosts(ctx)
			sw.syncVPPRatios(ctx)
			sw.syncFcapCounts(ctx)
		}
	}
}

func (sw *SettingsWatcher) syncFraudScoreBoosts(ctx context.Context) {
	rdb := sw.pickHealthyShard()
	if rdb == nil {
		return
	}

	newBoosts := make(map[uuid.UUID]uint8)
	prefix := "ml:score:boost:"

	for attempt := 0; attempt < len(sw.rdbs); attempt++ {
		cursor := uint64(0)
		ok := true
		for {
			keys, next, err := rdb.Scan(ctx, cursor, prefix+"*", 100).Result()
			if err != nil {
				slog.Warn("failed to scan ml boost keys from redis, trying next shard", "error", err)
				ok = false
				break
			}

			for _, key := range keys {
				parts := strings.Split(key, ":")
				if len(parts) < 4 {
					continue
				}
				campIDStr := parts[3]
				var campID uuid.UUID
				if !ParseUUID(UnsafeBytes(campIDStr), &campID) {
					continue
				}

				valStr, err := rdb.Get(ctx, key).Result()
				if err != nil {
					continue
				}
				val, err := strconv.Atoi(valStr)
				if err != nil {
					continue
				}
				if val < 0 {
					val = 0
				}
				if val > 100 {
					val = 100
				}
				newBoosts[campID] = uint8(val)
			}

			cursor = next
			if cursor == 0 {
				break
			}
		}
		if ok {
			sw.fraudScoreBoosts.Store(&FraudScoreBoostSnapshot{
				Boosts: newBoosts,
			})
			return
		}
		rdb = sw.nextShardAfter(rdb)
		if rdb == nil {
			return
		}
		newBoosts = make(map[uuid.UUID]uint8)
	}
}

func (sw *SettingsWatcher) syncVPPRatios(ctx context.Context) {
	rdb := sw.pickHealthyShard()
	if rdb == nil {
		return
	}

	newRatios := make(map[uuid.UUID]float32)
	prefix := "campaign:"
	suffix := ":pacing"

	for attempt := 0; attempt < len(sw.rdbs); attempt++ {
		cursor := uint64(0)
		ok := true
		for {
			keys, next, err := rdb.Scan(ctx, cursor, prefix+"*"+suffix, 100).Result()
			if err != nil {
				slog.Warn("failed to scan vpp pacing keys from redis", "error", err)
				ok = false
				break
			}
			for _, key := range keys {
				parts := strings.Split(key, ":")
				if len(parts) != 3 || parts[2] != "pacing" {
					continue
				}
				var campID uuid.UUID
				if !ParseUUID(UnsafeBytes(parts[1]), &campID) {
					continue
				}
				valStr, err := rdb.Get(ctx, key).Result()
				if err != nil {
					continue
				}
				val, err := strconv.ParseFloat(valStr, 32)
				if err != nil {
					continue
				}
				if val < 0 {
					val = 0
				}
				if val > 1 {
					val = 1
				}
				newRatios[campID] = float32(val)
			}
			cursor = next
			if cursor == 0 {
				break
			}
		}
		if ok {
			sw.vppRatios.Store(&VPPRatioSnapshot{Ratios: newRatios})
			return
		}
		rdb = sw.nextShardAfter(rdb)
		if rdb == nil {
			return
		}
		newRatios = make(map[uuid.UUID]float32)
	}
}

func (sw *SettingsWatcher) pickHealthyShard() redis.UniversalClient {
	if len(sw.rdbs) == 0 {
		return nil
	}
	for i := 1; i < len(sw.rdbs); i++ {
		if sw.rdbs[i] != nil {
			return sw.rdbs[i]
		}
	}
	return sw.rdbs[0]
}

func (sw *SettingsWatcher) nextShardAfter(cur redis.UniversalClient) redis.UniversalClient {
	if len(sw.rdbs) == 0 {
		return nil
	}
	found := false
	for _, rdb := range sw.rdbs {
		if found && rdb != nil && rdb != cur {
			return rdb
		}
		if rdb == cur {
			found = true
		}
	}
	for _, rdb := range sw.rdbs {
		if rdb != nil && rdb != cur {
			return rdb
		}
	}
	return nil
}

func (sw *SettingsWatcher) readConfigVersion(ctx context.Context) (int64, redis.UniversalClient, error) {
	for i, rdb := range sw.rdbs {
		if rdb == nil {
			continue
		}
		v, err := rdb.Get(ctx, "config:version").Int64()
		if err == nil {
			return v, rdb, nil
		}
		if err != redis.Nil {
			slog.Warn("failed to check config version on redis shard", "shard", i, "error", err)
		}
	}
	return 0, nil, redis.Nil
}

func (sw *SettingsWatcher) readConfigValues(ctx context.Context, preferred redis.UniversalClient) (map[string]string, error) {
	if preferred != nil {
		data, err := preferred.HGetAll(ctx, "config:values").Result()
		if err == nil {
			return data, nil
		}
	}
	for i, rdb := range sw.rdbs {
		if rdb == nil || rdb == preferred {
			continue
		}
		data, err := rdb.HGetAll(ctx, "config:values").Result()
		if err == nil {
			return data, nil
		}
		slog.Warn("failed to fetch config values on redis shard", "shard", i, "error", err)
	}
	return nil, redis.Nil
}

func (sw *SettingsWatcher) sync(ctx context.Context) {
	v, rdb, err := sw.readConfigVersion(ctx)
	if err != nil {
		if err != redis.Nil {
			slog.Error("failed to check config version on all redis shards", "error", err)
		}
		sw.trySyncFromPG(ctx)
		return
	}

	if v <= atomic.LoadInt64(&sw.currentVersion) {
		sw.trySyncFromPG(ctx)
		return
	}

	data, err := sw.readConfigValues(ctx, rdb)
	if err != nil {
		slog.Error("failed to fetch config values from redis", "error", err)
		sw.trySyncFromPG(ctx)
		return
	}

	sw.applyConfig(v, data)
}

func (sw *SettingsWatcher) trySyncFromPG(ctx context.Context) {
	if sw.pgSync == nil || sw.staleCheck == nil || !sw.staleCheck() {
		return
	}
	data, version, err := sw.pgSync(ctx)
	if err != nil {
		slog.Warn("settings pg fallback failed", "error", err)
		return
	}
	if version <= atomic.LoadInt64(&sw.currentVersion) {
		return
	}
	sw.applyConfig(version, data)
	slog.Info("dynamic settings updated from postgres (shard-0 degraded)", "version", version)
}

func (sw *SettingsWatcher) applyConfig(version int64, data map[string]string) {
	newCfg := sw.parseConfig(version, data)
	sw.snapshot.Store(newCfg)
	atomic.StoreInt64(&sw.currentVersion, version)

	for _, fn := range sw.onChange {
		fn(newCfg)
	}

	slog.Info("dynamic settings updated", "version", version)
}

func (sw *SettingsWatcher) parseConfig(version int64, data map[string]string) *DynamicConfig {
	current := sw.Get()
	next := *current
	next.Version = version

	updateInt(&next.RateLimitPerMin, data["rate_limit_per_min"])
	updateInt(&next.RateLimitWindow, data["rate_limit_window_ms"])
	updateMicro(&next.ClickAmount, data["click_amount"])
	updateMicro(&next.ImpressionAmount, data["impression_amount"])
	updateBool(&next.EmergencyBreaker, data["emergency_breaker"])
	updateInt(&next.FraudRLSuspectPct, data["fraud_rl_suspect_pct"])
	updateInt(&next.FraudRLIVTPct, data["fraud_rl_ivt_pct"])
	updateInt(&next.FraudRLBlockPct, data["fraud_rl_block_pct"])
	updateInt(&next.FraudRLRetrySuspect, data["fraud_rl_retry_suspect_sec"])
	updateInt(&next.FraudRLRetryIVT, data["fraud_rl_retry_ivt_sec"])
	updateInt(&next.FraudRLRetryBlock, data["fraud_rl_retry_block_sec"])
	updateString(&next.ASNCDNWhitelist, data["asn_cdn_whitelist"])
	updateString(&next.ASNMobileWhitelist, data["asn_mobile_whitelist"])
	updateString(&next.TLSHashBlocklist, data["tls_hash_blocklist"])
	updateString(&next.RtbBudgetAuthority, data[domain.SystemSettingRtbBudgetAuthority])
	updateString(&next.RtbMode, data[domain.SystemSettingRtbMode])

	return &next
}

func updateInt(target *int, val string) {
	if val == "" {
		return
	}
	if i, err := strconv.Atoi(val); err == nil {
		*target = i
	}
}

func updateMicro(target *int64, val string) {
	if val == "" {
		return
	}
	if micro, err := money.ParseDecimal(val); err == nil {
		*target = micro
	}
}

func updateBool(target *bool, val string) {
	if val == "" {
		return
	}
	if b, err := strconv.ParseBool(val); err == nil {
		*target = b
	}
}

func updateString(target *string, val string) {
	if val != "" {
		*target = val
	}
}
