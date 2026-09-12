package unified

import (
	"bytes"
	"context"
	_ "embed"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
	"unsafe"

	"ad-event-processor/internal/domain"
	filt "ad-event-processor/internal/filter"
	"ad-event-processor/internal/metrics"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"
)

type redisCmdHead struct {
	ctx         context.Context
	args        []any
	err         error
	keyPos      int8
	stepCount   int8
	rawVal      any
	readTimeout *time.Duration
	cmdType     redis.CmdType
	val         any
}

func resetPooledRedisCmd(cmd *redis.Cmd, ctx context.Context, args []any, firstKeyPos int8) {
	h := (*redisCmdHead)(unsafe.Pointer(cmd))
	h.ctx = ctx
	h.args = args
	h.err = nil
	h.keyPos = firstKeyPos
	h.rawVal = nil
	h.val = nil
}

func assignBufKey(dst *filt.StringVal, buf []byte) {
	if dst == nil {
		return
	}
	if len(buf) == 0 {
		dst.S = ""
		return
	}
	if len(dst.S) == len(buf) {
		if bytes.Equal(filt.UnsafeBytes(dst.S), buf) {
			return
		}
	}
	dst.S = filt.UnsafeString(buf)
}

func fillEvalShaWire(dst []any, sha1 any, keyArgs [unifiedFilterKeyCount]any, scriptArgs []any) []any {
	need := 3 + unifiedFilterKeyCount + len(scriptArgs)
	if cap(dst) < need {
		dst = make([]any, need, need+4)
	} else {
		dst = dst[:need]
	}
	dst[0] = evalShaCmdAny
	dst[1] = sha1
	dst[2] = numKeys15Any
	off := 3
	for i := range keyArgs {
		dst[off+i] = keyArgs[i]
	}
	off += unifiedFilterKeyCount
	for i := range scriptArgs {
		dst[off+i] = scriptArgs[i]
	}
	return dst
}

const evalShaWireCap = 3 + unifiedFilterKeyCount + 35

func (f *UnifiedFilter) evalShaPooled(ctx context.Context, c redis.UniversalClient, shard int, evt *domain.Event, sha1 any, keyArgs [unifiedFilterKeyCount]any, scriptArgs []any) (int64, error) {
	wire := fillEvalShaWire(f.evalFullWire[:], sha1, keyArgs, scriptArgs)

	resetPooledRedisCmd(&f.evalRedisCmd, ctx, wire, 3)
	if fc, ok := c.(filterEvalFastClient); ok {
		return fc.FilterEvalFast(&f.evalRedisCmd)
	}
	err := f.processFilterEval(ctx, c, shard, evt, &f.evalRedisCmd)
	val, intErr := f.evalRedisCmd.Int64()
	if intErr != nil && err == nil {
		err = intErr
	}
	if err != nil {
		return 0, err
	}
	return val, nil
}

func (f *UnifiedFilter) evalPooled(ctx context.Context, c redis.UniversalClient, shard int, evt *domain.Event, script any, keyArgs [unifiedFilterKeyCount]any, scriptArgs []any) (int64, error) {
	return f.evalPooledN(ctx, c, shard, evt, script, keyArgs[:], scriptArgs, unifiedFilterKeyCount)
}

func (f *UnifiedFilter) evalShaPooledN(ctx context.Context, c redis.UniversalClient, shard int, evt *domain.Event, sha1 any, keyArgs []any, scriptArgs []any, numKeys int) (int64, error) {
	wire := fillEvalShaWireN(f.evalFastWire[:], sha1, keyArgs, scriptArgs, numKeys)

	resetPooledRedisCmd(&f.evalRedisCmd, ctx, wire, 3)
	if fc, ok := c.(filterEvalFastClient); ok {
		return fc.FilterEvalFast(&f.evalRedisCmd)
	}
	err := f.processFilterEval(ctx, c, shard, evt, &f.evalRedisCmd)
	val, intErr := f.evalRedisCmd.Int64()
	if intErr != nil && err == nil {
		err = intErr
	}
	if err != nil {
		return 0, err
	}
	return val, nil
}

func (f *UnifiedFilter) evalPooledN(ctx context.Context, c redis.UniversalClient, shard int, evt *domain.Event, script any, keyArgs []any, scriptArgs []any, numKeys int) (int64, error) {
	wire := fillEvalWireN(f.evalFullWire[:], script, keyArgs, scriptArgs, numKeys)

	resetPooledRedisCmd(&f.evalRedisCmd, ctx, wire, 3)
	if fc, ok := c.(filterEvalFastClient); ok {
		return fc.FilterEvalFast(&f.evalRedisCmd)
	}
	err := f.processFilterEval(ctx, c, shard, evt, &f.evalRedisCmd)
	val, intErr := f.evalRedisCmd.Int64()
	if intErr != nil && err == nil {
		err = intErr
	}
	if err != nil {
		return 0, err
	}
	return val, nil
}

func fillEvalShaWireN(dst []any, sha1 any, keyArgs []any, scriptArgs []any, numKeys int) []any {
	need := 3 + numKeys + len(scriptArgs)
	if cap(dst) < need {
		dst = make([]any, need, need+4)
	} else {
		dst = dst[:need]
	}
	dst[0] = evalShaCmdAny
	dst[1] = sha1
	dst[2] = numKeysAny(numKeys)
	off := 3
	for i := range keyArgs {
		dst[off+i] = keyArgs[i]
	}
	off += numKeys
	for i := range scriptArgs {
		dst[off+i] = scriptArgs[i]
	}
	return dst
}

func fillEvalWireN(dst []any, script any, keyArgs []any, scriptArgs []any, numKeys int) []any {
	need := 3 + numKeys + len(scriptArgs)
	if cap(dst) < need {
		dst = make([]any, need, need+4)
	} else {
		dst = dst[:need]
	}
	dst[0] = evalCmdAny
	dst[1] = script
	dst[2] = numKeysAny(numKeys)
	off := 3
	for i := range keyArgs {
		dst[off+i] = keyArgs[i]
	}
	off += numKeys
	for i := range scriptArgs {
		dst[off+i] = scriptArgs[i]
	}
	return dst
}

func numKeysAny(n int) any {
	switch n {
	case 1:
		return numKeys1Any
	case unifiedFilterKeyCount:
		return numKeys19Any
	case budgetFastKeyCount:
		return numKeys9Any
	default:
		return n
	}
}

func isNoScriptErr(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, redis.ErrNoScript) {
		return true
	}
	return strings.Contains(err.Error(), "NOSCRIPT")
}

func (f *UnifiedFilter) PreloadScripts(ctx context.Context) error {
	if f == nil || f.script == nil || f.fastScript == nil || f.rollbackScript == nil {
		return fmt.Errorf("unified filter scripts are nil")
	}
	for i, redisClient := range f.redisShards {
		if redisClient == nil {
			continue
		}
		if err := f.preloadScriptsShard(ctx, i, redisClient); err != nil {
			return err
		}
	}
	return f.openFilterEvalPins(ctx)
}

func (f *UnifiedFilter) preloadScriptsShard(ctx context.Context, shard int, redisClient redis.UniversalClient) error {
	if f == nil || f.script == nil || f.fastScript == nil || f.rollbackScript == nil {
		return fmt.Errorf("unified filter scripts are nil")
	}
	if redisClient == nil {
		return fmt.Errorf("preload filter scripts shard %d: redis client is nil", shard)
	}
	shardLabel := strconv.Itoa(shard)
	if err := f.script.Load(ctx, redisClient).Err(); err != nil {
		metrics.RedisLuaScriptLoaded.WithLabelValues(shardLabel).Set(0)
		metrics.RedisLuaFastScriptLoaded.WithLabelValues(shardLabel).Set(0)
		return fmt.Errorf("preload filter full script shard %d: %w", shard, err)
	}
	if err := f.fastScript.Load(ctx, redisClient).Err(); err != nil {
		metrics.RedisLuaScriptLoaded.WithLabelValues(shardLabel).Set(0)
		metrics.RedisLuaFastScriptLoaded.WithLabelValues(shardLabel).Set(0)
		return fmt.Errorf("preload budget fast script shard %d: %w", shard, err)
	}
	if err := f.rollbackScript.Load(ctx, redisClient).Err(); err != nil {
		return fmt.Errorf("preload budget rollback script shard %d: %w", shard, err)
	}
	metrics.RedisLuaScriptLoaded.WithLabelValues(shardLabel).Set(1)
	metrics.RedisLuaFastScriptLoaded.WithLabelValues(shardLabel).Set(1)
	return nil
}

func (f *UnifiedFilter) AttachReconnectPreload() {
	if f == nil {
		return
	}
	for i, redisClient := range f.redisShards {
		if redisClient == nil {
			continue
		}
		redisClient.AddHook(newRedisShardPreloadHook(f, i))
	}
}

type redisShardPreloadHook struct {
	uf    *UnifiedFilter
	shard int
	mu    sync.Mutex
	last  time.Time
}

func newRedisShardPreloadHook(uf *UnifiedFilter, shard int) *redisShardPreloadHook {
	return &redisShardPreloadHook{uf: uf, shard: shard}
}

func (h *redisShardPreloadHook) DialHook(next redis.DialHook) redis.DialHook {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		conn, err := next(ctx, network, addr)
		if err == nil {
			h.schedulePreload(ctx)
		}
		return conn, err
	}
}

func (h *redisShardPreloadHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return next
}

func (h *redisShardPreloadHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return next
}

func (h *redisShardPreloadHook) schedulePreload(ctx context.Context) {
	if ctx == nil {
		return
	}
	h.mu.Lock()
	if time.Since(h.last) < time.Second {
		h.mu.Unlock()
		return
	}
	h.last = time.Now()
	filter := h.uf
	shard := h.shard
	h.mu.Unlock()

	go func(parent context.Context) {
		preloadCtx, cancel := context.WithTimeout(parent, 2*time.Second)
		defer cancel()
		if filter == nil || shard < 0 || shard >= len(filter.redisShards) {
			return
		}
		redisClient := filter.redisShards[shard]
		if redisClient == nil {
			return
		}
		if err := filter.preloadScriptsShard(preloadCtx, shard, redisClient); err != nil {
			slog.Warn("redis lua reconnect preload failed", "shard", shard, "error", err)
		}
	}(ctx)
}

// evalScript: primary unified-filter.lua path is EVALSHA; NOSCRIPT triggers bounded EVAL fallback
// (evalFallbackGate caps concurrent full-script loads) plus async PreloadScripts.
func (f *UnifiedFilter) evalScript(ctx context.Context, redisClient redis.UniversalClient, shard int, evt *domain.Event, keyArgs [unifiedFilterKeyCount]any, args []any) (int64, error) {
	res, err := f.evalShaPooled(ctx, redisClient, shard, evt, f.scriptHashAny, keyArgs, args)
	if err != nil && isNoScriptErr(err) {
		incRedisLuaNoScript(f.luaNoScriptCounters, shard)
		slog.Warn("redis lua NOSCRIPT encountered", "shard", shard, "error", err)

		go func(parent context.Context) {
			ctxPreheat, cancel := context.WithTimeout(parent, 2*time.Second)
			defer cancel()
			_ = f.PreloadScripts(ctxPreheat)
		}(ctx)

		if f.evalFallbackGate != nil {
			select {
			case f.evalFallbackGate <- struct{}{}:
				defer func() { <-f.evalFallbackGate }()
				return f.evalPooled(ctx, redisClient, shard, evt, unifiedFilterLuaAny, keyArgs, args)
			default:
				slog.Warn("redis lua NOSCRIPT fallback concurrency limit exceeded", "shard", shard)
				return -1, fmt.Errorf("redis lua EVAL fallback concurrency limit exceeded")
			}
		}
		return f.evalPooled(ctx, redisClient, shard, evt, unifiedFilterLuaAny, keyArgs, args)
	}
	return res, err
}

func PingConnectedRedisShards(ctx context.Context, redisShards []redis.UniversalClient) bool {
	if len(redisShards) == 0 {
		return true
	}
	checked := 0
	for i, redisClient := range redisShards {
		if redisClient == nil {
			continue
		}
		checked++
		if err := redisClient.Ping(ctx).Err(); err != nil {
			slog.Error("health check failed: redis shard", "shard", i, "error", err)
			return false
		}
	}
	return checked > 0
}

func firstConnectedRedisShard(redisShards []redis.UniversalClient) redis.UniversalClient {
	for _, redisClient := range redisShards {
		if redisClient != nil {
			return redisClient
		}
	}
	return nil
}

type RedisStreamTrimmerConfig struct {
	RedisShards  []redis.UniversalClient
	Streams      []string
	MaxLen       int
	TrimInterval time.Duration
}

type RedisStreamTrimmer struct {
	cfg    RedisStreamTrimmerConfig
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func NewRedisStreamTrimmer(cfg RedisStreamTrimmerConfig) *RedisStreamTrimmer {
	if cfg.MaxLen <= 0 {
		cfg.MaxLen = 10000
	}
	if cfg.TrimInterval <= 0 {
		cfg.TrimInterval = 10 * time.Second
	}
	return &RedisStreamTrimmer{
		cfg: cfg,
	}
}

func (t *RedisStreamTrimmer) Start(ctx context.Context) {
	if len(t.cfg.RedisShards) == 0 {
		return
	}
	runCtx, cancel := context.WithCancel(ctx)
	t.cancel = cancel

	t.wg.Add(1)
	go func() {
		defer t.wg.Done()

		t.TrimOnce(runCtx)

		ticker := time.NewTicker(t.cfg.TrimInterval)
		defer ticker.Stop()

		for {
			select {
			case <-runCtx.Done():
				return
			case <-ticker.C:
				t.TrimOnce(runCtx)
			}
		}
	}()
	slog.Info("redis stream trimmer started",
		"max_len", t.cfg.MaxLen,
		"trim_interval", t.cfg.TrimInterval.String(),
		"shards", len(t.cfg.RedisShards),
	)
}

func (t *RedisStreamTrimmer) TrimOnce(ctx context.Context) {
	for i, redisClient := range t.cfg.RedisShards {
		if redisClient == nil {
			continue
		}
		shardLabel := strconv.Itoa(i)

		for _, stream := range t.cfg.Streams {
			if stream == "" {
				continue
			}
			cmd := redisClient.XTrimMaxLenApprox(ctx, stream, int64(t.cfg.MaxLen), 0)
			if err := cmd.Err(); err != nil && !errors.Is(err, redis.Nil) {
				slog.Debug("redis stream xtrim error", "shard", i, "stream", stream, "error", err)
			}
		}

		infoCmd := redisClient.Info(ctx, "memory")
		if res, err := infoCmd.Result(); err == nil {
			if usedBytes := parseRedisUsedMemory(res); usedBytes >= 0 {
				metrics.RedisMemoryUsedBytes.WithLabelValues(shardLabel).Set(float64(usedBytes))
			}
		}
	}
}

func parseRedisUsedMemory(info string) int64 {
	info = strings.ReplaceAll(info, "\r", "")
	lines := strings.Split(info, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "used_memory:") {
			parts := strings.Split(line, ":")
			if len(parts) == 2 {
				if val, err := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64); err == nil {
					return val
				}
			}
		}
	}
	return -1
}

func (t *RedisStreamTrimmer) Close() {
	if t.cancel != nil {
		t.cancel()
	}
}

func (t *RedisStreamTrimmer) Wait() {
	t.wg.Wait()
}

//go:embed budget-fast.lua
var budgetFastLua string

var budgetFastLuaAny any

const (
	budgetFastKeyCount = 12
	budgetFastArgCount = 16
)

type budgetFastScratch struct {
	wIdem, wQuota, wFence, wFrozen, wFcapBump filt.BufWrapper
	args                                      [budgetFastArgCount]any
	wrappers                                  UnifiedStringWrappers
	keyVals                                   [budgetFastKeyCount]filt.StringVal
	keyArgs                                   [budgetFastKeyCount]any
}

func initBudgetFastScratch(s *budgetFastScratch) {
	if s == nil {
		return
	}
	if cap(s.wIdem.Buf) < 128 {
		s.wIdem.Buf = make([]byte, 0, 128)
		s.wQuota.Buf = make([]byte, 0, 128)
		s.wFence.Buf = make([]byte, 0, 128)
		s.wFrozen.Buf = make([]byte, 0, 128)
		s.wFcapBump.Buf = make([]byte, 0, 128)
	}
	for i := range s.keyVals {
		s.keyArgs[i] = &s.keyVals[i]
	}
}

type rollbackKeyScratch struct {
	wQuota    filt.BufWrapper
	wIdem     filt.BufWrapper
	wRollback filt.BufWrapper
}

var rollbackKeyScratchPool = sync.Pool{
	New: func() any {
		s := &rollbackKeyScratch{}
		s.wQuota.Buf = make([]byte, 0, 128)
		s.wIdem.Buf = make([]byte, 0, 128)
		s.wRollback.Buf = make([]byte, 0, 128)
		return s
	},
}

func appendClickRollbackGuardKey(dst []byte, campaignID uuid.UUID, clickID string) []byte {
	dst = filt.AppendCampaignHashTag(dst, campaignID)
	dst = append(dst, "rollback:click:"...)
	return append(dst, clickID...)
}

func (f *UnifiedFilter) runBudgetFastLua(
	ctx context.Context,
	evt *domain.Event,
	campInfo *domain.Campaign,
	amount any,
	redisClient redis.UniversalClient,
	shard int,
	scratch *budgetFastScratch,
) error {
	wIdem := &scratch.wIdem
	wQuota := &scratch.wQuota
	args := scratch.args[:]
	wrappers := &scratch.wrappers

	if campInfo == nil {
		return errors.New("budget fast: missing campaign")
	}
	budgetSourceKey := campInfo.BudgetCampaignKey
	subSlot := debitSubSlot(campInfo, evt.UserID, evt.ClickID)
	kv := scratch.keyVals[:]
	if f.quotaEnabledAny == oneAny {
		wQuota.Buf = appendBudgetQuotaKey(wQuota.Buf[:0], evt.CampaignID, subSlot)
		assignBufKey(&kv[0], wQuota.Buf)
		budgetSourceKey = kv[0].S
	}

	wIdem.Buf = wIdem.Buf[:0]
	wIdem.Buf = append(wIdem.Buf, "idempotency:click:"...)
	wIdem.Buf = append(wIdem.Buf, evt.ClickID...)
	assignBufKey(&kv[1], wIdem.Buf)
	idempotencyKey := kv[1].S

	wFence := &scratch.wFence
	wFence.Buf = wFence.Buf[:0]
	wFence.Buf = append(wFence.Buf, filt.MigrationFenceKeyPrefix...)
	wFence.Buf = filt.AppendUUID(wFence.Buf, evt.CampaignID)
	assignBufKey(&kv[7], wFence.Buf)
	migrationFenceKey := kv[7].S

	wFrozen := &scratch.wFrozen
	wFrozen.Buf = wFrozen.Buf[:0]
	wFrozen.Buf = append(wFrozen.Buf, filt.BudgetFrozenKeyPrefix...)
	wFrozen.Buf = filt.AppendUUID(wFrozen.Buf, evt.CampaignID)
	assignBufKey(&kv[8], wFrozen.Buf)
	budgetFrozenKey := kv[8].S

	kv[0].S = budgetSourceKey
	kv[1].S = idempotencyKey
	kv[2].S = campInfo.CampaignSyncKey
	kv[3].S = campInfo.CustomerSyncKey
	kv[7].S = migrationFenceKey
	kv[8].S = budgetFrozenKey
	keyArgs := scratch.keyArgs
	keyArgs[4] = &dirtyCampaignsKeyVal
	keyArgs[5] = &dirtyCustomersKeyVal
	// KEYS[7]: stream key or fcap:ignored; budget-fast.lua skips XADD when deferred to Go producer.
	keyArgs[6] = &f.streamKeyVal
	keyArgs[9] = &fcapIgnoredKeyVal
	fillLuaIgnoredPrecheckKeys(keyArgs[:], 11, 10)

	wrappers.clickID.S = evt.ClickID
	wrappers.evtType.S = evt.Type
	wrappers.payload.S = filt.UnsafeString(evt.Payload)
	wrappers.ip.S = evt.IP
	wrappers.ua.S = evt.UA
	wrappers.userID.S = evt.UserID

	args[0] = amount
	args[1] = f.idempotencyTTLAny
	args[2] = campInfo.IDStrAny
	args[3] = campInfo.CustomerIDStrAny
	args[4] = f.maxStreamLenAny
	args[5] = &wrappers.clickID
	args[6] = &wrappers.evtType
	args[7] = &wrappers.payload
	args[8] = &wrappers.ip
	args[9] = &wrappers.ua
	args[10] = &wrappers.userID
	args[11] = f.skipBudgetDebitAny
	args[12] = campInfo.LuaRoutingEpoch()
	args[13] = zeroAny
	args[14] = zeroAny
	args[15] = &wrappers.placementID
	wrappers.placementID.S = evt.PlacementID

	for i := range 2 {
		seq := f.luaMetricsSeq.Add(1)
		sampleLua := filt.ShouldSampleHistogram(seq, f.redisObservability.SampleMask())
		var luaStart int64
		if sampleLua || f.filterSlowNs > 0 {
			luaStart = filt.MonotonicNano()
		}
		f.redisObservability.RecordLuaOp(shard, evt.CampaignID, sampleLua)
		incRedisLuaTier(f.luaFastPathCounters, shard)
		res, err := f.evalFastScript(ctx, redisClient, shard, evt, keyArgs, args)
		f.noteLuaEvalDuration(shard, evt.CampaignID, "fast", luaStart, sampleLua, true)
		if err != nil {
			return err
		}
		if res == -1 {
			retry, recErr := f.recoverBudgetAfterMiss(ctx, evt, redisClient, budgetSourceKey, i)
			if recErr != nil {
				return recErr
			}
			if retry {
				continue
			}
			f.RecordShadowLuaOutcome(evt.CampaignID, true)
			return filt.ErrBudgetExhausted
		}
		if handled, handleErr := f.handleLuaResult(ctx, evt, campInfo, amount, redisClient, budgetSourceKey, shard, res, sampleLua); handled {
			if res == 3 {
				f.RecordShadowLuaOutcome(evt.CampaignID, true)
			}
			return handleErr
		}
	}
	return nil
}

func (f *UnifiedFilter) incLuaBranch(res int64) {
	if f == nil {
		return
	}
	switch res {
	case 0:
		f.filterLuaBranchOK.Inc()
	case 2:
		f.filterLuaBranchDuplicate.Inc()
	default:
		metrics.FilterLuaBranchTotal.WithLabelValues(luaBranchLabel(res)).Inc()
	}
}

func (f *UnifiedFilter) handleLuaResult(
	ctx context.Context,
	evt *domain.Event,
	campInfo *domain.Campaign,
	amount any,
	redisClient redis.UniversalClient,
	budgetSourceKey string,
	shard int,
	res int64,
	sampleLua bool,
) (handled bool, err error) {
	if res == -1 {
		return false, nil
	}

	f.incLuaBranch(res)

	switch res {
	case 1:
		return true, filt.ErrRateLimitExceeded
	case 2:
		return true, filt.ErrDuplicateEvent
	case 3:
		if f.quotaMode == "live" {
			f.localQuotaCache.Block(evt.CampaignID, filt.MonotonicNano())
		}
		return true, filt.ErrBudgetExhausted
	case 4:
		return true, filt.ErrPacingExhausted
	case 5:
		return true, filt.ErrFreqLimitExceeded
	case 6:
		filt.AddFraudSignal(evt, filt.FraudReasonLowTTC)
		return true, nil
	case 7:
		filt.AddFraudSignal(evt, filt.FraudReasonMissingImpTS)
		return true, nil
	case 10:
		filt.AddFraudSignal(evt, filt.FraudReasonMissingImpTS)
		metrics.TTCBypassTotal.Inc()
		if ttcEnabled(f.ttcMinMsAny) {
			debitAmount := spendMicroFromAny(amount)
			if debitAmount > 0 && campInfo != nil {
				f.RollbackDebit(ctx, evt, campInfo, debitAmount, false)
			}
			return true, filt.ErrFilterTimeout
		}
		metrics.EventsProcessed.Inc()
		f.recordAcceptedSpendIfDebited(shard, evt.CampaignID, amount, sampleLua)
		return true, nil
	case 11:
		return true, filt.ErrMigrationFenced
	case luaReturnDailyQuota:
		return true, filt.ErrDailyQuotaExceeded
	case luaReturnFraudSignal:
		filt.AddFraudSignal(evt, filt.FraudReasonL3Blocklist)
		metrics.EventsProcessed.Inc()
		f.recordAcceptedSpendIfDebited(shard, evt.CampaignID, amount, sampleLua)
		return true, nil
	case luaReturnPlacement:
		return true, filt.ErrPlacementBlocked
	case luaReturnTierDegraded:
		metrics.FilterTierDegradedTotal.Inc()
		debitAmount := spendMicroFromAny(amount)
		if debitAmount > 0 && campInfo != nil {
			f.RollbackDebit(ctx, evt, campInfo, debitAmount, false)
		}
		return true, filt.ErrFilterTimeout
	default:
		// Lua debit succeeded; ingest publishAcceptedTrack runs after Check returns nil.
		// Post-debit producer reject must call RollbackDebit (not handled inside this package).
		metrics.EventsProcessed.Inc()
		f.recordAcceptedSpendIfDebited(shard, evt.CampaignID, amount, sampleLua)
		return true, nil
	}
}

func (f *UnifiedFilter) evalFastScript(ctx context.Context, redisClient redis.UniversalClient, shard int, evt *domain.Event, keyArgs [budgetFastKeyCount]any, args []any) (int64, error) {
	// Fast-path debit-only script; same EVALSHA / NOSCRIPT fallback contract as evalScript.
	res, err := f.evalShaPooledN(ctx, redisClient, shard, evt, f.fastScriptHashAny, keyArgs[:], args, budgetFastKeyCount)
	if err != nil && isNoScriptErr(err) {
		incRedisLuaNoScript(f.luaNoScriptCounters, shard)
		slog.Warn("redis lua NOSCRIPT encountered (fast script)", "shard", shard, "error", err)

		go func(parent context.Context) {
			ctxPreheat, cancel := context.WithTimeout(parent, 2*time.Second)
			defer cancel()
			_ = f.PreloadScripts(ctxPreheat)
		}(ctx)

		if f.evalFallbackGate != nil {
			select {
			case f.evalFallbackGate <- struct{}{}:
				defer func() { <-f.evalFallbackGate }()
				return f.evalPooledN(ctx, redisClient, shard, evt, budgetFastLuaAny, keyArgs[:], args, budgetFastKeyCount)
			default:
				slog.Warn("redis lua NOSCRIPT fallback concurrency limit exceeded (fast script)", "shard", shard)
				return -1, fmt.Errorf("redis lua EVAL fallback concurrency limit exceeded")
			}
		}
		return f.evalPooledN(ctx, redisClient, shard, evt, budgetFastLuaAny, keyArgs[:], args, budgetFastKeyCount)
	}
	return res, err
}

func (f *UnifiedFilter) recoverBudgetAfterMiss(
	ctx context.Context,
	evt *domain.Event,
	redisClient redis.UniversalClient,
	budgetSourceKey string,
	attempt int,
) (retry bool, err error) {
	metrics.BudgetCacheMissTotal.Inc()
	if attempt > 0 {
		return false, filt.ErrBudgetExhausted
	}
	// Budget key miss recovery respects evt.FilterDeadlineMono; no Postgres warm past deadline.
	if filt.FilterDeadlineExceededEvt(evt, ctx) {
		return false, filt.ErrFilterTimeout
	}

	worker := -1
	if evt != nil {
		worker = int(evt.FilterWorkerIdx)
	}
	recovered, recErr := filt.TryRecoverBudgetFromRegistry(ctx, redisClient, f.registry, evt.CampaignID, budgetSourceKey, worker)
	if recErr != nil {
		return false, recErr
	}
	if recovered {
		return true, nil
	}

	if !f.postgresFallbackAllowed {
		return false, filt.ErrBudgetExhausted
	}

	dbTimeout := f.dbLookupTimeout
	if rem, ok := filt.FilterDeadlineRemainingEvt(evt, ctx); ok {
		if rem <= 0 {
			return false, filt.ErrFilterTimeout
		}
		if rem < dbTimeout {
			dbTimeout = rem
		}
	}

	metrics.BudgetCacheMissPostgresTotal.Inc()
	if f.repo == nil {
		return false, filt.ErrBudgetExhausted
	}
	dbCtx, cancel := context.WithTimeout(ctx, dbTimeout)
	camp, err := f.repo.GetByID(dbCtx, evt.CampaignID)
	cancel()
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return false, filt.ErrFilterTimeout
		}
		return false, err
	}

	remaining := camp.BudgetLimit - camp.CurrentSpend
	if remaining < 0 {
		remaining = 0
	}
	if err := filt.WarmBudgetKeyNX(ctx, redisClient, budgetSourceKey, remaining); err != nil {
		return false, err
	}
	return true, nil
}

//go:embed budget-rollback.lua
var budgetRollbackLua string

// RollbackRedisDebit runs budget-rollback.lua (EVALSHA) after ingest post-debit enqueue failure
// on the Redis debit path; restores budget key, idempotency lock, and dirty sync sets.
func (f *UnifiedFilter) RollbackRedisDebit(
	ctx context.Context,
	evt *domain.Event,
	campInfo *domain.Campaign,
	amount int64,
) error {
	if f == nil || evt == nil || campInfo == nil {
		return nil
	}

	shard, _, err := f.resolveDebitShard(evt.CampaignID, evt.UserID, evt.ClickID, campInfo)
	if err != nil {
		return fmt.Errorf("rollback: failed to resolve shard: %w", err)
	}
	redisClient := f.redisShards[shard%len(f.redisShards)]
	if redisClient == nil {
		return fmt.Errorf("rollback: redis client is nil for shard %d", shard)
	}

	scratch := rollbackKeyScratchPool.Get().(*rollbackKeyScratch)
	defer rollbackKeyScratchPool.Put(scratch)

	budgetSourceKey := campInfo.BudgetCampaignKey
	if f.quotaEnabledAny == oneAny {
		subSlot := debitSubSlot(campInfo, evt.UserID, evt.ClickID)
		scratch.wQuota.Buf = appendBudgetQuotaKey(scratch.wQuota.Buf[:0], evt.CampaignID, subSlot)
		budgetSourceKey = filt.UnsafeString(scratch.wQuota.Buf)
	}

	scratch.wIdem.Buf = scratch.wIdem.Buf[:0]
	scratch.wIdem.Buf = append(scratch.wIdem.Buf, "idempotency:click:"...)
	scratch.wIdem.Buf = append(scratch.wIdem.Buf, evt.ClickID...)
	idempotencyKey := filt.UnsafeString(scratch.wIdem.Buf)

	scratch.wRollback.Buf = appendClickRollbackGuardKey(scratch.wRollback.Buf[:0], evt.CampaignID, evt.ClickID)
	rollbackGuardKey := filt.UnsafeString(scratch.wRollback.Buf)

	keys := []string{
		budgetSourceKey,
		idempotencyKey,
		campInfo.CampaignSyncKey,
		campInfo.CustomerSyncKey,
		dirtyCampaignsKeyVal.S,
		dirtyCustomersKeyVal.S,
		rollbackGuardKey,
	}

	args := []any{
		amount,
		campInfo.ID.String(),
		campInfo.CustomerID.String(),
		f.idempotencyTTLAny,
	}

	var code int64
	code, err = redisClient.EvalSha(ctx, f.rollbackScriptHash, keys, args...).Int64()
	if err != nil && isNoScriptErr(err) {
		code, err = redisClient.Eval(ctx, budgetRollbackLua, keys, args...).Int64()
	}

	if err != nil {
		slog.Error("failed to rollback redis debit",
			"campaign_id", evt.CampaignID,
			"click_id", evt.ClickID,
			"amount", amount,
			"error", err,
		)
		return err
	}

	switch code {
	case 1:
		slog.Info("successfully rolled back redis debit",
			"campaign_id", evt.CampaignID,
			"click_id", evt.ClickID,
			"amount", amount,
		)
		return nil
	case 2:
		slog.Debug("rollback redis debit skipped: already applied",
			"campaign_id", evt.CampaignID,
			"click_id", evt.ClickID,
			"amount", amount,
		)
		return nil
	case 0:
		return fmt.Errorf("rollback: invalid amount %d", amount)
	default:
		return fmt.Errorf("rollback: unexpected lua code %d", code)
	}
}

// RollbackDebit dispatches by debit source: local quanta ledger refund when evt.LocalQuantaDebitMicro
// is pending, otherwise RollbackRedisDebit. isLocalQuanta is set by FilterEngine from pending debit only.
func (f *UnifiedFilter) RollbackDebit(
	ctx context.Context,
	evt *domain.Event,
	campInfo *domain.Campaign,
	amount int64,
	isLocalQuanta bool,
) {
	if f == nil || evt == nil || campInfo == nil {
		return
	}
	_ = isLocalQuanta
	if evt.LocalQuantaDebitMicro > 0 {
		amountMicro := evt.LocalQuantaDebitMicro
		evt.LocalQuantaDebitMicro = 0
		if f.localRollbackGuard != nil && !f.localRollbackGuard.TryClaim(evt.ClickID) {
			slog.Debug("rollback local quanta skipped: already applied",
				"campaign_id", evt.CampaignID,
				"click_id", evt.ClickID,
				"amount", amountMicro,
			)
			return
		}
		subSlot := debitSubSlot(campInfo, evt.UserID, evt.ClickID)
		f.rollbackLocalQuantaSpend(evt.CampaignID, subSlot, amountMicro)
		f.releaseLocalQuantaClickIdem(evt.ClickID)
		f.rollbackLocalFcapForEvent(evt)
		metrics.LocalQuotaRollbackTotal.Inc()
		return
	}
	f.rollbackLocalFcapForEvent(evt)
	_ = f.RollbackRedisDebit(ctx, evt, campInfo, amount)
}

func observeRedisLua(observers []prometheus.Observer, shard int, seconds float64) {
	if shard >= 0 && shard < len(observers) {
		observers[shard].Observe(seconds)
		return
	}
	metrics.RedisLuaDuration.WithLabelValues(strconv.Itoa(shard)).Observe(seconds)
}

func newRedisLuaTierObservers(numShards int) []prometheus.Observer {
	if numShards <= 0 {
		numShards = 1
	}
	observers := make([]prometheus.Observer, numShards)
	for i := range observers {
		observers[i] = metrics.RedisLuaFastDuration.WithLabelValues(strconv.Itoa(i))
	}
	return observers
}

func newRedisLuaPathCounters(numShards int, fast bool) []prometheus.Counter {
	if numShards <= 0 {
		numShards = 1
	}
	counters := make([]prometheus.Counter, numShards)
	for i := range counters {
		shard := strconv.Itoa(i)
		if fast {
			counters[i] = metrics.RedisLuaFastPathTotal.WithLabelValues(shard)
		} else {
			counters[i] = metrics.RedisLuaFullPathTotal.WithLabelValues(shard)
		}
	}
	return counters
}

func incRedisLuaTier(counters []prometheus.Counter, shard int) {
	if shard >= 0 && shard < len(counters) {
		counters[shard].Inc()
		return
	}
	metrics.RedisLuaFastPathTotal.WithLabelValues(strconv.Itoa(shard)).Inc()
}

func observeRedisLuaTier(observers []prometheus.Observer, shard int, seconds float64) {
	if shard >= 0 && shard < len(observers) {
		observers[shard].Observe(seconds)
		return
	}
	metrics.RedisLuaFastDuration.WithLabelValues(strconv.Itoa(shard)).Observe(seconds)
}

func incRedisLuaNoScript(counters []prometheus.Counter, shard int) {
	if shard >= 0 && shard < len(counters) {
		counters[shard].Inc()
		return
	}
	metrics.RedisLuaNoScriptTotal.WithLabelValues(strconv.Itoa(shard)).Inc()
}
