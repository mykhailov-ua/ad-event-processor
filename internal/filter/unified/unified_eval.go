package unified

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"ad-event-processor/internal/domain"
	filt "ad-event-processor/internal/filter"
	"ad-event-processor/internal/licensing"
	"ad-event-processor/internal/metrics"
	"ad-event-processor/internal/telemetry"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type JSONSerializationFilter struct {
	registry domain.CampaignRegistry
	enabled  bool
}

func NewJSONSerializationFilter(registry domain.CampaignRegistry) *JSONSerializationFilter {
	return &JSONSerializationFilter{registry: registry}
}

func (f *JSONSerializationFilter) SetEnabled(enabled bool) {
	if f == nil {
		return
	}
	f.enabled = enabled
}

func (f *JSONSerializationFilter) Check(ctx context.Context, evt *domain.Event) error {
	if f == nil || !f.enabled || evt == nil || evt.JSONSerializationFlags == 0 {
		return nil
	}
	if f.registry == nil {
		return nil
	}
	camp, ok := f.registry.GetCampaign(evt.CampaignID)
	if !ok || camp == nil || !camp.JSONSerializationEnabled {
		return nil
	}
	metrics.JSONSerializationBotTotal.Inc()
	filt.AddFraudSignal(evt, filt.FraudReasonJSONSerializationBot)
	return nil
}

type L7WireFilter struct {
	secFetchEnabled         atomic.Bool
	clientHintsEnabled      atomic.Bool
	tlsALPNEnabled          atomic.Bool
	h2SettingsEnabled       atomic.Bool
	h2PseudoOrderEnabled    atomic.Bool
	h2DowngradeEnabled      atomic.Bool
	http1HeaderOrderEnabled atomic.Bool
	acceptEncodingEnabled   atomic.Bool
}

func NewL7WireFilter() *L7WireFilter {
	f := &L7WireFilter{}
	f.secFetchEnabled.Store(true)
	f.clientHintsEnabled.Store(true)
	f.tlsALPNEnabled.Store(true)
	f.h2SettingsEnabled.Store(true)
	f.h2PseudoOrderEnabled.Store(true)
	f.h2DowngradeEnabled.Store(true)
	f.http1HeaderOrderEnabled.Store(true)
	f.acceptEncodingEnabled.Store(true)
	return f
}

func (f *L7WireFilter) SetSecFetchEnabled(enabled bool) {
	f.secFetchEnabled.Store(enabled)
}

func (f *L7WireFilter) SetClientHintsPlatformEnabled(enabled bool) {
	f.clientHintsEnabled.Store(enabled)
}

func (f *L7WireFilter) SetTLSALPNMismatchEnabled(enabled bool) {
	f.tlsALPNEnabled.Store(enabled)
}

func (f *L7WireFilter) SetH2SettingsEnabled(enabled bool) {
	f.h2SettingsEnabled.Store(enabled)
}

func (f *L7WireFilter) SetH2PseudoOrderEnabled(enabled bool) {
	f.h2PseudoOrderEnabled.Store(enabled)
}

func (f *L7WireFilter) SetH2DowngradeEnabled(enabled bool) {
	f.h2DowngradeEnabled.Store(enabled)
}

func (f *L7WireFilter) SetHTTP1HeaderOrderEnabled(enabled bool) {
	f.http1HeaderOrderEnabled.Store(enabled)
}

func (f *L7WireFilter) SetAcceptEncodingEnabled(enabled bool) {
	f.acceptEncodingEnabled.Store(enabled)
}

func (f *L7WireFilter) Check(ctx context.Context, evt *domain.Event) error {
	if f == nil || evt == nil {
		return nil
	}
	if f.secFetchEnabled.Load() && filt.SecFetchAnomaly(evt.UA, evt.SecFetchPresent, evt.SecFetchMode, evt.SecFetchDest) {
		filt.AddFraudSignal(evt, filt.FraudReasonSecFetchAnomaly)
	}
	if f.clientHintsEnabled.Load() && filt.ClientHintsPlatformMismatch(evt.UA, evt.SecCHUAPlatform, evt.SecCHUAMobile) {
		filt.AddFraudSignal(evt, filt.FraudReasonClientHintsMismatch)
	}
	if f.tlsALPNEnabled.Load() && filt.TLSALPNBrowserMismatch(evt.UA, evt.TLSALPN) {
		filt.AddFraudSignal(evt, filt.FraudReasonTLSALPNMismatch)
	}
	if evt.IngressH2 != 0 {
		if f.h2SettingsEnabled.Load() && filt.H2SettingsAnomaly(evt.UA, evt.H2WireFlags, evt.H2EnablePush, evt.H2InitialWindow, evt.H2WindowUpdateInc) {
			filt.AddFraudSignal(evt, filt.FraudReasonH2SettingsMismatch)
		}
		if f.h2PseudoOrderEnabled.Load() && filt.H2PseudoOrderMismatch(evt.UA, evt.H2PseudoOrder, evt.H2PseudoOrderCount) {
			filt.AddFraudSignal(evt, filt.FraudReasonH2PseudoOrder)
		}
		if f.h2DowngradeEnabled.Load() && filt.H2DowngradeArtifact(evt.H2WireFlags) {
			filt.AddFraudSignal(evt, filt.FraudReasonH2DowngradeArtifact)
		}
	} else if f.http1HeaderOrderEnabled.Load() && filt.HTTP1HeaderOrderMismatch(evt.UA, evt.HTTP1HeaderOrder[:], evt.HTTP1HeaderOrderCount, evt.SecFetchPresent) {
		filt.AddFraudSignal(evt, filt.FraudReasonHeaderOrderMismatch)
	}
	if f.acceptEncodingEnabled.Load() && filt.AcceptEncodingBrowserMismatch(evt.UA, evt.AcceptEncodingFlags, evt.AcceptEncodingSet) {
		filt.AddFraudSignal(evt, filt.FraudReasonAcceptEncodingMismatch)
	}
	return nil
}

type TCPMSSFilter struct {
	minMSS           uint8
	tunnelEnabled    bool
	tunnelThreshold  uint16
	mobileCarrierASN *filt.MobileCarrierASNTable
	dcASN            *filt.DCASNTable
	asnLookup        filt.ASNLookup
}

func NewTCPMSSFilter(minMSS uint8) *TCPMSSFilter {
	if minMSS == 0 {
		minMSS = 2
	}
	return &TCPMSSFilter{minMSS: minMSS}
}

func (f *TCPMSSFilter) ConfigureTunnel(
	enabled bool,
	threshold uint16,
	mobileCarrierASN *filt.MobileCarrierASNTable,
	dcASN *filt.DCASNTable,
	lookup filt.ASNLookup,
) {
	if f == nil {
		return
	}
	if threshold == 0 {
		threshold = 1400
	}
	f.tunnelEnabled = enabled
	f.tunnelThreshold = threshold
	f.mobileCarrierASN = mobileCarrierASN
	f.dcASN = dcASN
	f.asnLookup = lookup
}

func (f *TCPMSSFilter) Check(ctx context.Context, evt *domain.Event) error {
	if f == nil || evt == nil || evt.TCPMSSSet == 0 {
		return nil
	}
	if tcpMSSHighByte(evt.TCPMSS) < f.minMSS {
		metrics.TCPMSSAnomalyTotal.WithLabelValues("low_mss").Inc()
		filt.AddFraudSignal(evt, filt.FraudReasonTCPMSSAnomaly)
	}
	f.checkTunnel(evt)
	return nil
}

func (f *TCPMSSFilter) checkTunnel(evt *domain.Event) {
	if !f.tunnelEnabled || f.asnLookup == nil || evt.IP == "" {
		return
	}
	if tcpMSSWireValue(evt.TCPMSS) >= f.tunnelThreshold {
		return
	}
	asn, ok := f.asnLookup.LookupASN(evt.IP)
	if !ok || asn == 0 {
		return
	}
	if f.mobileCarrierASN != nil && f.mobileCarrierASN.IsMobileCarrier(asn) {
		return
	}
	if f.dcASN != nil && f.dcASN.Ready() && f.dcASN.IsDatacenter(asn) {
		return
	}
	metrics.TCPMSSAnomalyTotal.WithLabelValues("tunnel_mss").Inc()
	filt.AddFraudSignal(evt, filt.FraudReasonTCPTunnelMSS)
}

func tcpMSSHighByte(mss uint16) uint8 {
	if mss <= 255 {
		return uint8(mss)
	}
	return uint8(mss >> 8)
}

func tcpMSSWireValue(mss uint16) uint16 {
	if mss <= 255 {
		return mss << 8
	}
	return mss
}

type QuantaLedger interface {
	HasCredit(id uuid.UUID) bool
	TrySpendDebit(id uuid.UUID, subSlot int, amountMicro int64) bool
	RefundDebit(id uuid.UUID, subSlot int, amountMicro int64)
	Remaining(id uuid.UUID) int64
	SetMode(mode string)
}

type QuantaStrictGate interface {
	IsStrict(id uuid.UUID) bool
	UpdateFromRedisRemaining(id uuid.UUID, redisRemaining int64)
}

type QuantaRefillSignaler interface {
	Signal(campaignID uuid.UUID)
}

type QuantaDeltaPublisher interface {
	Publish(campaignID uuid.UUID, amountMicro int64)
	PublishReturn(campaignID uuid.UUID, amountMicro int64)
}

type QuantaClickIdem interface {
	TryClaim(clickID string) bool
	Release(clickID string)
}

type QuantaStreamPublisher interface {
	Enqueue(shard int, evt *domain.Event, camp *domain.Campaign, amountMicro int64) bool
	SetStreamName(name string)
}

type LocalQuantaDeps struct {
	Ledger    QuantaLedger
	Strict    QuantaStrictGate
	Refill    QuantaRefillSignaler
	Publisher QuantaDeltaPublisher
	Stream    QuantaStreamPublisher
	Idem      QuantaClickIdem
}

func (f *UnifiedFilter) SetLocalQuantaDeps(deps LocalQuantaDeps) {
	f.localQuantaLedger = deps.Ledger
	f.localQuantaStrict = deps.Strict
	f.localQuantaRefill = deps.Refill
	f.localQuantaPublisher = deps.Publisher
	f.localQuantaStream = deps.Stream
	f.localClickIdem = deps.Idem
}

func (f *UnifiedFilter) SetLocalQuantaMode(mode string) {
	f.localQuotaMode = mode
	if f.localQuantaLedger != nil {
		f.localQuantaLedger.SetMode(mode)
	}
}

func (f *UnifiedFilter) localQuantaActive() bool {
	return f.localQuotaMode == "shadow" || f.localQuotaMode == "live"
}

func (f *UnifiedFilter) LocalQuantaEligible(evt *domain.Event, campInfo *domain.Campaign) bool {
	return f.localQuantaEligible(evt, campInfo)
}

func (f *UnifiedFilter) localQuantaEligible(evt *domain.Event, campInfo *domain.Campaign) bool {
	if f.localQuantaLedger == nil || !f.localQuantaActive() {
		return false
	}
	if f.quotaEnabledAny != oneAny {
		return false
	}
	if f.localQuantaStrict != nil && f.localQuantaStrict.IsStrict(evt.CampaignID) {
		return false
	}
	if !f.fastPathEnabled.Load() || f.needsFullLuaPath(evt, campInfo) {
		return false
	}
	if evt.Type != "impression" && evt.Type != "click" {
		return false
	}
	return true
}

func (f *UnifiedFilter) localQuantaFullSkipEligible(evt *domain.Event, campInfo *domain.Campaign) bool {
	if f.localQuotaMode != "live" || f.localQuantaStream == nil {
		return false
	}
	if !f.localQuantaEligible(evt, campInfo) {
		return false
	}
	return true
}

func (f *UnifiedFilter) checkLocalQuanta(
	ctx context.Context,
	evt *domain.Event,
	campInfo *domain.Campaign,
	amountMicro int64,
) (handled bool, err error) {
	if !f.localQuantaEligible(evt, campInfo) {
		return false, nil
	}

	amount := amountMicro
	if amount <= 0 {
		if evt.Type == "impression" {
			amount = f.impressionAmountMicro
		} else {
			amount = f.clickAmountMicro
		}
	}

	if f.localQuotaMode == "shadow" {
		subSlot := debitSubSlot(campInfo, evt.UserID, evt.ClickID)
		localOK := f.localQuantaLedger.TrySpendDebit(evt.CampaignID, subSlot, amount)
		if localOK {
			f.publishLocalDelta(evt.CampaignID, amount)
		}
		return false, nil
	}

	if campInfo.FreqLimit > 0 && evt.UserID != "" {
		exceeded, err := f.checkFreqLimitGo(evt, campInfo)
		if err != nil {
			return true, err
		}
		if exceeded {
			return true, filt.ErrFreqLimitExceeded
		}
	}

	subSlot := debitSubSlot(campInfo, evt.UserID, evt.ClickID)
	if !f.localQuantaLedger.TrySpendDebit(evt.CampaignID, subSlot, amount) {
		if f.localQuantaRefill != nil {
			f.localQuantaRefill.Signal(evt.CampaignID)
		}
		return false, nil
	}

	metrics.LocalQuotaSpendTotal.Inc()
	f.publishLocalDelta(evt.CampaignID, amount)

	if f.localQuantaFullSkipEligible(evt, campInfo) {
		metrics.LocalQuotaFullSkipEligibleTotal.Inc()
		err := f.acceptLocalQuantaFullSkip(ctx, evt, campInfo, amount, subSlot)
		return true, err
	}

	shard, _, err := f.resolveDebitShard(evt.CampaignID, evt.UserID, evt.ClickID, campInfo)
	if err != nil {
		f.rollbackLocalQuantaSpend(evt.CampaignID, subSlot, amount)
		return true, err
	}
	redisClient := f.redisShards[shard%len(f.redisShards)]

	debitAny := f.clickAmountMicroAny
	if evt.Type == "impression" {
		debitAny = f.impressionAmountMicroAny
	}

	prevSkip := f.skipBudgetDebitAny
	f.skipBudgetDebitAny = oneAny
	fastScratch := budgetFastScratchPool.Get().(*budgetFastScratch)
	err = f.runBudgetFastLua(ctx, evt, campInfo, debitAny, redisClient, shard, fastScratch)
	f.skipBudgetDebitAny = prevSkip
	budgetFastScratchPool.Put(fastScratch)

	if err != nil {
		f.rollbackLocalQuantaSpend(evt.CampaignID, subSlot, amount)
		return true, err
	}
	return true, nil
}

func (f *UnifiedFilter) rollbackLocalQuantaSpend(campaignID uuid.UUID, subSlot int, amountMicro int64) {
	if f.localQuantaLedger != nil && amountMicro > 0 {
		f.localQuantaLedger.RefundDebit(campaignID, subSlot, amountMicro)
	}
	if f.localQuantaPublisher != nil && amountMicro > 0 {
		f.localQuantaPublisher.PublishReturn(campaignID, amountMicro)
	}
}

func (f *UnifiedFilter) AcceptLocalQuantaFullSkip(ctx context.Context, evt *domain.Event, campInfo *domain.Campaign, amountMicro int64, subSlot int) error {
	return f.acceptLocalQuantaFullSkip(ctx, evt, campInfo, amountMicro, subSlot)
}

func (f *UnifiedFilter) acceptLocalQuantaFullSkip(ctx context.Context, evt *domain.Event, campInfo *domain.Campaign, amountMicro int64, subSlot int) error {
	if f.localClickIdem != nil && !f.localClickIdem.TryClaim(evt.ClickID) {
		metrics.FilterLuaBranchTotal.WithLabelValues("duplicate").Inc()
		f.rollbackLocalQuantaSpend(evt.CampaignID, subSlot, amountMicro)
		return filt.ErrDuplicateEvent
	}

	shard, _, err := f.resolveDebitShard(evt.CampaignID, evt.UserID, evt.ClickID, campInfo)
	if err != nil {
		if f.localClickIdem != nil {
			f.localClickIdem.Release(evt.ClickID)
		}
		f.rollbackLocalQuantaSpend(evt.CampaignID, subSlot, amountMicro)
		return err
	}

	if !f.localQuantaStream.Enqueue(shard, evt, campInfo, amountMicro) {
		if f.localClickIdem != nil {
			f.localClickIdem.Release(evt.ClickID)
		}
		f.rollbackLocalQuantaSpend(evt.CampaignID, subSlot, amountMicro)
		return filt.ErrShardUnavailable
	}

	metrics.LocalQuotaFullSkipTotal.Inc()
	metrics.RedisLuaSkippedTotal.Inc()
	metrics.EventsProcessed.Inc()
	telemetry.RecordAccepted()
	return nil
}

func (f *UnifiedFilter) publishLocalDelta(campaignID uuid.UUID, amountMicro int64) {
	if f.localQuantaPublisher != nil {
		f.localQuantaPublisher.Publish(campaignID, amountMicro)
	}
}

func (f *UnifiedFilter) RecordShadowLuaOutcome(campaignID uuid.UUID, luaBudgetExhausted bool) {
	if f.localQuotaMode != "shadow" || f.localQuantaLedger == nil {
		return
	}
	localHad := f.localQuantaLedger.Remaining(campaignID) >= 0 && f.localQuantaLedger.HasCredit(campaignID)
	if localHad && luaBudgetExhausted {
		metrics.LocalQuotaShadowDiffTotal.Inc()
	}
}

func (f *UnifiedFilter) UpdateStrictFromRedis(campaignID uuid.UUID, redisRemaining int64) {
	if f.localQuantaStrict != nil {
		f.localQuantaStrict.UpdateFromRedisRemaining(campaignID, redisRemaining)
	}
}

type ResidentialProxyFilter struct {
	ring         *filt.ResidentialProxyRing
	intelTable   *filt.ResidentialIntelTable
	intelEnabled bool
}

func NewResidentialProxyFilter(ring *filt.ResidentialProxyRing) *ResidentialProxyFilter {
	if ring == nil {
		return &ResidentialProxyFilter{}
	}
	return &ResidentialProxyFilter{ring: ring}
}

func (f *ResidentialProxyFilter) SetIntelTable(table *filt.ResidentialIntelTable, enabled bool) {
	if f == nil {
		return
	}
	f.intelTable = table
	f.intelEnabled = enabled && table != nil
}

func (f *ResidentialProxyFilter) Check(ctx context.Context, evt *domain.Event) error {
	if f == nil || evt == nil {
		return nil
	}
	if f.intelEnabled && f.intelTable.Ready() && f.intelTable.MatchIP(evt.IP) {
		metrics.ResidentialIntelHotMatchTotal.Inc()
		metrics.ResidentialProxySignalTotal.Inc()
		filt.AddFraudSignal(evt, filt.FraudReasonResidentialProxy)
		return nil
	}
	if f.ring == nil {
		return nil
	}
	isClick := evt.Type == "click"
	userHash := filt.HashResidentialProxyUser(evt.UserID)
	uaHash := filt.HashResidentialProxyUA(evt.UA)
	campaignHash := filt.CRC32Castagnoli(&evt.CampaignID)
	_, signal := f.ring.Observe(campaignHash, isClick, userHash, uaHash, filt.MonotonicNano())
	if signal {
		metrics.ResidentialProxySignalTotal.Inc()
		filt.AddFraudSignal(evt, filt.FraudReasonResidentialProxy)
	}
	return nil
}

const (
	LuaReturnDailyQuota   int64 = 12
	LuaReturnPlacement    int64 = 14
	LuaReturnTierDegraded int64 = 20
	LuaReturnFraudSignal  int64 = 21

	luaReturnDailyQuota   = LuaReturnDailyQuota
	luaReturnPlacement    = LuaReturnPlacement
	luaReturnTierDegraded = LuaReturnTierDegraded
	luaReturnFraudSignal  = LuaReturnFraudSignal

	luaPrecheckIngressTTLSec = 28 * 3600
	luaDegradeThresholdNs    = int64(2_000_000)
)

func LuaBranchLabel(res int64) string {
	return luaBranchLabel(res)
}

func luaBranchLabel(res int64) string {
	switch res {
	case 0:
		return "ok"
	case 1:
		return "rate"
	case 2:
		return "duplicate"
	case 3:
		return "budget"
	case 4:
		return "pacing"
	case 5:
		return "freq"
	case 6:
		return "ttc_low"
	case 7:
		return "ttc_missing"
	case 10:
		return "ttc_bypass"
	case 11:
		return "migration_fence"
	case luaReturnDailyQuota:
		return "daily_quota"
	case luaReturnPlacement:
		return "placement"
	case luaReturnTierDegraded:
		return "tier_degraded"
	case luaReturnFraudSignal:
		return "fraud_signal"
	default:
		return "accept"
	}
}

var luaDegradeThresholdAny any = luaDegradeThresholdNs

var (
	placementIgnoredKeyVal = filt.StringVal{S: "fcap:ignored"}
	ingressIgnoredKeyVal   = filt.StringVal{S: "fcap:ignored"}
)

type entitlementsLookup interface {
	GetEntitlements(customerID uuid.UUID) (licensing.Entitlements, bool)
}

func (f *UnifiedFilter) entitlementsMaxRPD(custID uuid.UUID) uint64 {
	lookup, ok := f.registry.(entitlementsLookup)
	if !ok {
		return 0
	}
	ent, ok := lookup.GetEntitlements(custID)
	if !ok || ent.Limits.MaxRequestsPerDay == 0 {
		return 0
	}
	return ent.Limits.MaxRequestsPerDay
}

func appendCampaignIngressDayKey(dst []byte, campaignID uuid.UUID, regionCode uint8, customerID uuid.UUID, t time.Time) []byte {
	dst = filt.AppendCampaignHashTag(dst[:0], campaignID)
	dst = append(dst, "ingress:day:"...)
	if regionCode > 0 {
		dst = append(dst, hexByte(regionCode>>4), hexByte(regionCode&0x0f), ':')
	}
	dst = filt.AppendUUID(dst, customerID)
	dst = append(dst, ':')
	return filt.AppendDate(dst, t)
}

func (f *UnifiedFilter) SetFraudBlacklistFilter(bl *filt.FraudBlacklistFilter) {
	if f != nil {
		f.fraudBL = bl
	}
}

func (f *UnifiedFilter) SetIngressRPDHandledExternally(v bool) {
	if f != nil {
		f.ingressRPDHandledExternally = v
	}
}

func (f *UnifiedFilter) ConfigureCGNAT(globalBypass bool, table *filt.MobileCarrierASNTable, lookup filt.ASNLookup) {
	if f == nil {
		return
	}
	f.cgnatGlobalBypass = globalBypass
	f.mobileCarrierASN = table
	f.asnLookup = lookup
}

func (f *UnifiedFilter) ApplyLuaGoPrechecks(
	ctx context.Context,
	evt *domain.Event,
	campInfo *domain.Campaign,
	redisClient redis.UniversalClient,
	now time.Time,
) error {
	return f.applyLuaGoPrechecks(ctx, evt, campInfo, redisClient, now)
}

func (f *UnifiedFilter) applyLuaGoPrechecks(
	ctx context.Context,
	evt *domain.Event,
	campInfo *domain.Campaign,
	redisClient redis.UniversalClient,
	now time.Time,
) error {
	if f.ingressRPDHandledExternally {
		return nil
	}
	return f.checkIngressRPDGo(ctx, evt, campInfo, redisClient, now)
}

func (f *UnifiedFilter) checkIngressRPDGo(
	ctx context.Context,
	evt *domain.Event,
	campInfo *domain.Campaign,
	redisClient redis.UniversalClient,
	now time.Time,
) error {
	maxRPD := f.entitlementsMaxRPD(campInfo.CustomerID)
	if maxRPD == 0 || redisClient == nil {
		return nil
	}
	if filt.CgnatBypassForCampaign(f.cgnatGlobalBypass, f.registry, evt.CampaignID, f.mobileCarrierASN, f.asnLookup, evt.IP, "ingress_rpd") {
		return nil
	}
	var keyBuf []byte
	keyBuf = appendCampaignIngressDayKey(keyBuf, evt.CampaignID, f.regionCode, campInfo.CustomerID, now)
	redisKey := filt.UnsafeString(keyBuf)
	pipe := redisClient.Pipeline()
	incr := pipe.Incr(ctx, redisKey)
	pipe.Expire(ctx, redisKey, time.Duration(luaPrecheckIngressTTLSec)*time.Second)
	if _, err := pipe.Exec(ctx); err != nil {
		return nil
	}
	if uint64(incr.Val()) > maxRPD {
		return filt.ErrDailyQuotaExceeded
	}
	return nil
}

func fillLuaIgnoredPrecheckKeys(keyArgs []any, ingressIdx, placementIdx int) {
	keyArgs[ingressIdx] = &ingressIgnoredKeyVal
	keyArgs[placementIdx] = &placementIgnoredKeyVal
}

func quotaRefillSample(campaignID uuid.UUID) bool {
	return campaignID[0]%100 == 0
}

func ttcEnabled(ttcMinMsAny any) bool {
	if ttcMinMsAny == nil || ttcMinMsAny == zeroAny {
		return false
	}
	switch v := ttcMinMsAny.(type) {
	case int64:
		return v > 0
	case int:
		return v > 0
	default:
		return false
	}
}

func (f *UnifiedFilter) NeedsFullLuaPath(evt *domain.Event, campInfo *domain.Campaign) bool {
	return f.needsFullLuaPath(evt, campInfo)
}

func (f *UnifiedFilter) needsFullLuaPath(evt *domain.Event, campInfo *domain.Campaign) bool {
	if evt.Type != "impression" && evt.Type != "click" {
		return true
	}
	if !f.fastPathEnabled.Load() {
		return true
	}
	if campInfo.FreqLimit > 0 && evt.UserID != "" {
		if f.settingsWatcher == nil {
			return true
		}
	}
	if campInfo.PacingMode == domain.PacingModeEven {
		if f.roughPacing == nil || !campInfo.RoughPacingEnabled() {
			return true
		}
	}
	if ttcEnabled(f.ttcMinMsAny) && f.localTTC == nil {
		return true
	}
	if f.quotaEnabledAny == oneAny && quotaRefillSample(evt.CampaignID) && f.localQuantaRefill == nil {
		return true
	}
	return false
}

type filterEvalPinSlot struct {
	client *redis.Client
	conn   *redis.Conn
}

type filterEvalPin struct {
	shards  int
	workers int
	slots   []filterEvalPinSlot
}

func (p *filterEvalPin) slot(worker, shard int) *filterEvalPinSlot {
	if p == nil || worker < 0 || worker >= p.workers || shard < 0 || shard >= p.shards {
		return nil
	}
	return &p.slots[worker*p.shards+shard]
}

func (p *filterEvalPin) conn(worker, shard int) *redis.Conn {
	s := p.slot(worker, shard)
	if s == nil {
		return nil
	}
	return s.conn
}

func (f *UnifiedFilter) SetFilterEvalPinWorkers(workers int) {
	if f == nil {
		return
	}
	if workers < 0 {
		workers = 0
	}
	f.evalPinWorkers = workers
}

func (f *UnifiedFilter) FilterEvalPinWorkers() int {
	if f == nil {
		return 0
	}
	return f.evalPinWorkers
}

func (f *UnifiedFilter) evalPinConn(evt *domain.Event, shard int) *redis.Conn {
	if f == nil || f.evalPins == nil || evt == nil || evt.FilterWorkerIdx < 0 {
		return nil
	}
	worker := int(evt.FilterWorkerIdx)
	if worker >= f.evalPinWorkers {
		return nil
	}
	return f.evalPins.conn(worker, shard)
}

func isStickyConnRetryable(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, redis.ErrClosed) || errors.Is(err, io.EOF) {
		return true
	}
	s := err.Error()
	return strings.Contains(s, "closed") ||
		strings.Contains(s, "EOF") ||
		strings.Contains(s, "bad state") ||
		strings.Contains(s, "broken pipe") ||
		strings.Contains(s, "reset by peer")
}

func (f *UnifiedFilter) processFilterEval(ctx context.Context, c redis.UniversalClient, shard int, evt *domain.Event, cmd redis.Cmder) error {
	pin := f.evalPinConn(evt, shard)
	err := processRedisCmd(ctx, c, pin, cmd)
	if err == nil || pin == nil || evt == nil || evt.FilterWorkerIdx < 0 {
		return err
	}
	if !isStickyConnRetryable(err) {
		return err
	}
	worker := int(evt.FilterWorkerIdx)
	if reopenErr := f.reopenEvalPin(ctx, worker, shard); reopenErr != nil {
		return err
	}
	pin = f.evalPinConn(evt, shard)
	return processRedisCmd(ctx, c, pin, cmd)
}

func (f *UnifiedFilter) openFilterEvalPins(ctx context.Context) error {
	if f == nil || f.evalPinWorkers <= 0 || len(f.redisShards) == 0 {
		return nil
	}
	workers := f.evalPinWorkers
	shards := len(f.redisShards)
	pin := &filterEvalPin{
		workers: workers,
		shards:  shards,
		slots:   make([]filterEvalPinSlot, workers*shards),
	}
	for w := range workers {
		for i, redisClient := range f.redisShards {
			client, ok := redisClient.(*redis.Client)
			if !ok {
				continue
			}
			slot := pin.slot(w, i)
			slot.client = client
			slot.conn = client.Conn()
			if err := slot.conn.Ping(ctx).Err(); err != nil {
				f.closeFilterEvalPins()
				return fmt.Errorf("ping filter eval pin worker=%d shard=%d: %w", w, i, err)
			}
		}
	}
	f.evalPins = pin
	return nil
}

func (f *UnifiedFilter) reopenEvalPin(ctx context.Context, worker, shard int) error {
	if f == nil || f.evalPins == nil {
		return fmt.Errorf("filter eval pins not open")
	}
	slot := f.evalPins.slot(worker, shard)
	if slot == nil || slot.client == nil {
		return fmt.Errorf("filter eval pin worker=%d shard=%d unavailable", worker, shard)
	}
	if slot.conn != nil {
		_ = slot.conn.Close()
		slot.conn = nil
	}
	slot.conn = slot.client.Conn()
	if err := slot.conn.Ping(ctx).Err(); err != nil {
		return err
	}
	return nil
}

func (f *UnifiedFilter) CloseFilterEvalPins() {
	if f == nil {
		return
	}
	f.closeFilterEvalPins()
}

func (f *UnifiedFilter) closeFilterEvalPins() {
	if f == nil || f.evalPins == nil {
		return
	}
	for i := range f.evalPins.slots {
		if f.evalPins.slots[i].conn != nil {
			_ = f.evalPins.slots[i].conn.Close()
			f.evalPins.slots[i].conn = nil
		}
	}
	f.evalPins = nil
}

func processRedisCmd(ctx context.Context, c redis.UniversalClient, pin *redis.Conn, cmd redis.Cmder) error {
	if pin != nil {
		return pin.Process(ctx, cmd)
	}
	return c.Process(ctx, cmd)
}

const unifiedFilterKeyCount = 19

var (
	evalShaCmdAny any = "evalsha"
	evalCmdAny    any = "eval"
	numKeys15Any  any = unifiedFilterKeyCount
	numKeys19Any  any = unifiedFilterKeyCount
	numKeys9Any   any = budgetFastKeyCount
	numKeys1Any   any = 1
)

var evalWirePool = sync.Pool{
	New: func() any {
		s := make([]any, 56, 64)
		return &s
	},
}

var evalCmdPool = sync.Pool{
	New: func() any {
		return redis.NewCmd(context.Background())
	},
}
