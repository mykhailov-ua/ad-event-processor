package ingest

import (
	"context"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/filter"
	"ad-event-processor/internal/rtb"

	"github.com/jackc/pgx/v5/pgxpool"
)

type (
	BudgetAuthority              = rtb.BudgetAuthority
	RtbCatalog                   = rtb.RtbCatalog
	RtbCampaignInput             = rtb.RtbCampaignInput
	RtbTargetingInput            = rtb.RtbTargetingInput
	RtbBudgetMirrorWriter        = rtb.RtbBudgetMirrorWriter
	RtbBudgetReconcileConfig     = rtb.RtbBudgetReconcileConfig
	RtbBudgetReconcileWorker     = rtb.RtbBudgetReconcileWorker
	RtbBudgetSync                = rtb.RtbBudgetSync
	DealFloorCache               = rtb.DealFloorCache
	RtbDealOutcomeWriter         = rtb.RtbDealOutcomeWriter
	RtbExchangeLogWriter         = rtb.RtbExchangeLogWriter
	RtbShadowDiffSnapshotDTO     = rtb.RtbShadowDiffSnapshotDTO
	RtbLiveGateResult            = rtb.RtbLiveGateResult
	SupplyChainAllowlistSnapshot = rtb.SupplyChainAllowlistSnapshot
	SchainNodes                  = rtb.SchainNodes
	SchainNode                   = rtb.SchainNode
)

const (
	BudgetAuthorityRedis  = rtb.BudgetAuthorityRedis
	BudgetAuthorityRTB    = rtb.BudgetAuthorityRTB
	BudgetAuthorityShadow = rtb.BudgetAuthorityShadow
	CTRPPMUnit            = rtb.CTRPPMUnit
)

var (
	ErrInvalidRtbBudgetAuthority         = rtb.ErrInvalidRtbBudgetAuthority
	NewRtbCatalog                        = rtb.NewRtbCatalog
	NewRtbBudgetMirrorWriter             = rtb.NewRtbBudgetMirrorWriter
	NewRtbBudgetReconcileWorker          = rtb.NewRtbBudgetReconcileWorker
	NewDealFloorCache                    = rtb.NewDealFloorCache
	NewRtbDealOutcomeWriter              = rtb.NewRtbDealOutcomeWriter
	NewRtbExchangeLogWriter              = rtb.NewRtbExchangeLogWriter
	SetRtbDealOutcomeWriter              = rtb.SetRtbDealOutcomeWriter
	SetRtbExchangeLogWriter              = rtb.SetRtbExchangeLogWriter
	BudgetAuthorityFromSettings          = rtb.BudgetAuthorityFromSettings
	BudgetAuthorityFromConfig            = rtb.BudgetAuthorityFromConfig
	RtbSkipLuaBudgetDebit                = rtb.RtbSkipLuaBudgetDebit
	NormalizeRtbBudgetAuthoritySetting   = rtb.NormalizeRtbBudgetAuthoritySetting
	SyncRtbCatalog                       = rtb.SyncRtbCatalog
	StartRtbCatalogSync                  = rtb.StartRtbCatalogSync
	StartRtbCatalogReloadWatch           = rtb.StartRtbCatalogReloadWatch
	ReloadRtbCatalog                     = rtb.ReloadRtbCatalog
	BuildRtbInputsFromRegistry           = rtb.BuildRtbInputsFromRegistry
	BuildRtbCatalogRows                  = rtb.BuildRtbCatalogRows
	BidRequestFromEvent                  = rtb.BidRequestFromEvent
	CampaignDataFromDomain               = rtb.CampaignDataFromDomain
	CampaignIDFromUUID                   = rtb.CampaignIDFromUUID
	GeoHashFromCountry                   = rtb.GeoHashFromCountry
	GeoHashFromCountryBytes              = rtb.GeoHashFromCountryBytes
	DeviceMaskFromType                   = rtb.DeviceMaskFromType
	RtbShadowDiffForWindow               = rtb.RtbShadowDiffForWindow
	ResetRtbShadowDiffBuckets            = rtb.ResetRtbShadowDiffBuckets
	EvaluateRtbLiveGate                  = rtb.EvaluateRtbLiveGate
	CanEnableRtbLive                     = rtb.CanEnableRtbLive
	ValidateSchainNodes                  = rtb.ValidateSchainNodes
	BuildSupplyChainAllowlistFromSellers = rtb.BuildSupplyChainAllowlistFromSellers
	LoadSupplyChainAllowlist             = rtb.LoadSupplyChainAllowlist
	StartDealFloorRefresh                = rtb.StartDealFloorRefresh
	EffectiveDealFloor                   = rtb.EffectiveDealFloor
	EffectiveDealFloorBytes              = rtb.EffectiveDealFloorBytes
	RtbModeFromSetting                   = rtb.RtbModeFromSetting
	NormalizeRtbModeSetting              = rtb.NormalizeRtbModeSetting
	HybridMaxRPSFromConfig               = rtb.HybridMaxRPSFromConfig
	BuildCampaignMetaList                = rtb.BuildCampaignMetaList
	SyncRTBBudgetState                   = rtb.SyncRTBBudgetState
)

const (
	rtbModeOff    = rtb.RtbModeOff
	rtbModeShadow = rtb.RtbModeShadow
	rtbModeLive   = rtb.RtbModeLive
)

var (
	recordRtbDealOutcomeBytes = rtb.RecordRtbDealOutcomeBytes
	recordRtbShadowAuction    = rtb.RecordRtbShadowAuction
	recordRtbExchangeLog      = rtb.RecordRtbExchangeLog
)

type rtbExchangeLogMeta = rtb.RtbExchangeLogMeta

const (
	rtbExchangeRequestIDMax = rtb.RtbExchangeRequestIDMax
	rtbExchangeBidIDMax     = rtb.RtbExchangeBidIDMax
	rtbExchangeDealIDMax    = rtb.RtbExchangeDealIDMax
)

var (
	NoBidInvalidRequest = rtb.NoBidInvalidRequest
	NoBidNoCandidates   = rtb.NoBidNoCandidates
	NewBudgetStore      = rtb.NewBudgetStore
	rtbModeFromConfig   = rtb.RtbModeFromConfig
)

func fraudBoostsFromWatcher(watcher *SettingsWatcher) *rtb.FraudBoostSnapshot {
	if watcher == nil {
		return nil
	}
	snap := watcher.GetFraudScoreBoosts()
	if snap == nil {
		return nil
	}
	return &rtb.FraudBoostSnapshot{Boosts: snap.Boosts}
}

func noBidToRejectKind(reason rtb.NoBidReason) filter.FilterRejectKind {
	switch reason {
	case rtb.NoBidPacingClosed:
		return filterRejectPacing
	case rtb.NoBidDaypartClosed:
		return filterRejectSchedule
	case rtb.NoBidFreqCapExceeded:
		return filterRejectFreq
	case rtb.NoBidDailyCapExceeded, rtb.NoBidSpendFailed:
		return filterRejectBudget
	case rtb.NoBidNoCandidates, rtb.NoBidEmptyShard:
		return filterRejectBidFloor
	case rtb.NoBidCorruptCatalog, rtb.NoBidInvalidRequest:
		return filterRejectInfra
	default:
		return filterRejectBidFloor
	}
}

type RtbAuthorityController struct {
	cfg        *config.Config
	watcher    *SettingsWatcher
	unified    *UnifiedFilter
	catalog    *RtbCatalog
	budgetSync *RtbBudgetSync
}

func NewRtbAuthorityController(
	cfg *config.Config,
	watcher *SettingsWatcher,
	unified *UnifiedFilter,
	catalog *RtbCatalog,
	budgetSync *RtbBudgetSync,
) *RtbAuthorityController {
	c := &RtbAuthorityController{
		cfg:        cfg,
		watcher:    watcher,
		unified:    unified,
		catalog:    catalog,
		budgetSync: budgetSync,
	}
	if watcher != nil {
		watcher.AddChangeListener(func(_ *DynamicConfig) { c.Apply() })
	}
	c.Apply()
	return c
}

func (c *RtbAuthorityController) Apply() {
	setting := ""
	if c.watcher != nil {
		setting = c.watcher.Get().RtbBudgetAuthority
	}
	auth := BudgetAuthorityFromSettings(c.cfg, setting)
	if c.unified != nil {
		c.unified.SetSkipBudgetDebit(RtbSkipLuaBudgetDebit(c.cfg, setting))
	}
	if c.catalog != nil {
		c.catalog.SetAuthority(auth)
	}
	if c.budgetSync != nil {
		c.budgetSync.Authority = auth
	}
}

func RunRtbBidShadeSim(ctx context.Context, pool *pgxpool.Pool, cfg *config.Config, in domain.RtbBidShadeInput) (domain.RtbBidShadeOutput, error) {
	out := domain.RtbBidShadeOutput{}
	if pool == nil {
		out.NoBidReason = NoBidInvalidRequest.String()
		return out, nil
	}
	registry := NewRegistry(db.New(pool))
	if _, err := registry.Sync(ctx); err != nil {
		return out, err
	}
	if cfg == nil {
		cfg = &config.Config{ClickAmount: 1, ImpressionAmount: 1}
	}
	catalog := NewRtbCatalog(NewBudgetStore(), BudgetAuthorityShadow)
	catalog.Registry().SetTargetingIndexEnabled(cfg.RtbTargetingIndexEnabled())
	SyncRtbCatalog(ctx, registry, catalog, cfg, nil, RtbBudgetSync{}, nil)

	targeting := RtbTargetingInput{
		GeoHash:             in.GeoHash,
		DeviceType:          in.DeviceType,
		CategoryMask:        in.CategoryMask,
		PublisherFloorMicro: in.MinBidMicro,
	}
	if targeting.CategoryMask == 0 {
		targeting.CategoryMask = 1
	}
	bidReq := BidRequestFromEvent(nil, targeting)
	res, reason := catalog.Registry().RunAuctionEval(&bidReq)
	if !reason.OK() {
		out.NoBidReason = reason.String()
		return out, nil
	}
	uid, ok := catalog.UUIDForWinner(res.CampaignID)
	if !ok {
		out.NoBidReason = NoBidNoCandidates.String()
		return out, nil
	}
	out.HasBid = true
	out.CampaignID = uid.String()
	out.ClearingPriceMicro = res.Price
	out.RecommendedBidMicro = res.Price - res.Price/50
	if out.RecommendedBidMicro < in.MinBidMicro {
		out.RecommendedBidMicro = in.MinBidMicro
	}
	out.ShadeDeltaMicro = res.Price - out.RecommendedBidMicro
	if res.Price > 0 {
		out.SecondPriceDeltaPct = float64(out.ShadeDeltaMicro) * 100 / float64(res.Price)
	}
	return out, nil
}

func ConfigureTrackRtb(proc *trackProcessor, cfg *config.Config, catalog *RtbCatalog, geo GeoProvider, unified *UnifiedFilter, watcher *SettingsWatcher) {
	if proc == nil || cfg == nil || catalog == nil || !cfg.RtbEnabled() {
		return
	}
	if !openRTBLicenseAllowed(proc.registry) {
		proc.rtbMode = rtbModeOff
		return
	}
	proc.rtbCatalog = catalog
	proc.rtbMode = rtbModeFromConfig(cfg)
	proc.settingsWatcher = watcher
	if watcher != nil {
		proc.rtbMode = RtbModeFromSetting(watcher.Get().RtbMode, cfg)
		watcher.AddChangeListener(func(dc *DynamicConfig) {
			if proc != nil {
				proc.rtbMode = RtbModeFromSetting(dc.RtbMode, cfg)
			}
		})
	}
	proc.ingestGeo = geo
	catalog.ConfigureRtbGates(watcher, geo)
	if cfg.RtbPrebidIVTEnabled() {
		catalog.SetPrebidIVT(true)
	}
	if unified != nil {
		setting := ""
		if watcher != nil {
			setting = watcher.Get().RtbBudgetAuthority
		}
		unified.SetSkipBudgetDebit(RtbSkipLuaBudgetDebit(cfg, setting))
	}
}

func ConfigureIngestGeo(proc *trackProcessor, geo GeoProvider) {
	if proc != nil {
		proc.ingestGeo = geo
	}
}

func (h *AdsPacketHandler) ConfigureRtb(catalog *RtbCatalog, geo GeoProvider, unified *UnifiedFilter, watcher *SettingsWatcher) {
	if h == nil {
		return
	}
	ConfigureTrackRtb(&h.trackProc, h.cfg, catalog, geo, unified, watcher)
}

func (h *AdsPacketHandler) ConfigureIngestGeo(geo GeoProvider) {
	if h == nil {
		return
	}
	ConfigureIngestGeo(&h.trackProc, geo)
}

func buildRtbTargeting(evt *domain.Event, deviceType []byte, floorMicro int64, catalog *RtbCatalog) RtbTargetingInput {
	geoHash := uint32(0)
	if evt != nil && evt.IngestGeoResolved {
		geoHash = evt.GeoHash
	}

	out := RtbTargetingInput{GeoHash: geoHash}

	if evt != nil && len(evt.Payload) > 0 {
		var parsed OpenRTB3Parsed
		var haveParsed bool
		if cached, ok := openRTB3ParsedFromScratch(evt); ok {
			parsed = *cached
			haveParsed = true
		} else {
			parsed = parseOpenRTB3FSM(evt.Payload)
			haveParsed = parsed.IsOpenRTB
		}
		if haveParsed {
			if floorMicro <= 0 {
				floorMicro = parsed.MinBid
			}
			if parsed.DealIDLen > 0 {
				out.DealIDLen = parsed.DealIDLen
				src := ortbSlice(evt.Payload, parsed.DealIDOff, parsed.DealIDLen)
				copy(out.DealIDBuf[:], src)
			}
			floorMicro = EffectiveDealFloorBytes(catalog, catalogDealFloors(catalog), out.DealIDBuf[:out.DealIDLen], floorMicro)
			out.DeviceType = parsed.DeviceType
			out.CategoryMask = parsed.CategoryMask
			out.PublisherFloorMicro = floorMicro
			return out
		}
	}

	if evt != nil && len(evt.Payload) > 0 {
		incIngressLegacyJSON()
	}
	if floorMicro <= 0 && evt != nil {
		floorMicro = parseBidMicro(evt.Payload)
	}
	categoryMask := uint64(1)
	if evt != nil {
		if parsed := parseCategoryMask(evt.Payload); parsed != 0 {
			categoryMask = parsed
		}
		if n := ParseDealIDBytes(evt.Payload, out.DealIDBuf[:]); n > 0 {
			out.DealIDLen = uint8(n)
		}
	}
	if out.DealIDLen > 0 {
		floorMicro = EffectiveDealFloorBytes(catalog, catalogDealFloors(catalog), out.DealIDBuf[:out.DealIDLen], floorMicro)
	}
	out.DeviceType = DeviceMaskFromType(deviceType)
	out.CategoryMask = categoryMask
	out.PublisherFloorMicro = floorMicro
	return out
}

func catalogDealFloors(catalog *RtbCatalog) *DealFloorCache {
	if catalog == nil {
		return nil
	}
	return catalog.DealFloors
}

func applyRtbAuction(proc trackProcessor, evt *domain.Event, deviceType []byte) (trackOutcome, bool) {
	if proc.rtbCatalog == nil || proc.rtbMode == rtbModeOff || evt == nil {
		return trackOutcome{}, false
	}
	if !openRTBLicenseAllowed(proc.registry) {
		return trackOutcome{}, false
	}

	targeting := buildRtbTargeting(evt, deviceType, 0, proc.rtbCatalog)
	payloadBidMicro := targeting.PublisherFloorMicro
	res, reason := proc.rtbCatalog.RunAuction(evt, targeting)

	if proc.rtbMode == rtbModeShadow {
		recordRtbShadowAuction(proc.rtbCatalog, evt, res, reason, payloadBidMicro)
		recordRtbDealOutcomeBytes(targeting.DealIDBuf[:], targeting.DealIDLen, payloadBidMicro, res, reason)
		return trackOutcome{}, false
	}

	recordRtbDealOutcomeBytes(targeting.DealIDBuf[:], targeting.DealIDLen, payloadBidMicro, res, reason)

	if !reason.OK() {
		return trackOutcome{Status: trackStatusRejected, RejectKind: noBidToRejectKind(reason)}, true
	}

	uid, ok := proc.rtbCatalog.UUIDForWinner(res.CampaignID)
	if !ok {
		return trackOutcome{Status: trackStatusRejected, RejectKind: filterRejectCampaignNotFound}, true
	}
	evt.CampaignID = uid
	evt.ClearingPriceMicro = res.Price
	return trackOutcome{}, false
}
