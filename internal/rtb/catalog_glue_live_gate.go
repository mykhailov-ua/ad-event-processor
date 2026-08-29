package rtb

import (
	"context"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/metrics"
	"ad-event-processor/internal/openrtb"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

type RtbLiveGateResult struct {
	Ready         bool                     `json:"ready"`
	Reasons       []string                 `json:"reasons,omitempty"`
	Shadow        RtbShadowDiffSnapshotDTO `json:"shadow"`
	ReconcileHigh bool                     `json:"reconcile_high"`
}

func EvaluateRtbLiveGate(window time.Duration) RtbLiveGateResult {
	if window <= 0 {
		window = rtbLiveGateDefaultWindow
	}
	out := RtbLiveGateResult{
		Shadow: RtbShadowDiffForWindow(window),
	}
	out.ReconcileHigh = rtbBudgetReconcileHigh()
	var reasons []string
	if out.Shadow.ShadowEvals < rtbLiveGateMinShadowEvals {
		reasons = append(reasons, rtbLiveGateInsufficient)
	} else if out.Shadow.ParityRate < rtbLiveGateMinParityRate {
		reasons = append(reasons, rtbLiveGateMismatchReason)
	}
	if out.ReconcileHigh {
		reasons = append(reasons, rtbLiveGateReconcileReason)
	}
	out.Reasons = reasons
	out.Ready = len(reasons) == 0
	return out
}

func rtbBudgetReconcileHigh() bool {
	metric := &dto.Metric{}
	if err := metrics.RtbBudgetReconcileHigh.Write(metric); err != nil {
		return false
	}
	return metric.GetGauge().GetValue() >= 1
}

func CanEnableRtbLive(window time.Duration) (bool, []string) {
	gate := EvaluateRtbLiveGate(window)
	return gate.Ready, gate.Reasons
}

const SystemSettingRtbMode = domain.SystemSettingRtbMode

var ErrInvalidRtbMode = domain.ErrInvalidRtbMode

func NormalizeRtbModeSetting(v string) (string, error) {
	return domain.NormalizeRtbModeSetting(v)
}

func RtbModeFromSetting(setting string, cfg *config.Config) uint8 {
	raw := strings.TrimSpace(setting)
	if raw == "" && cfg != nil {
		return RtbModeFromConfig(cfg)
	}
	switch config.ParseRtbMode(raw) {
	case config.RtbModeShadow:
		return RtbModeShadow
	case config.RtbModeLive:
		return RtbModeLive
	default:
		return RtbModeOff
	}
}

func rtbPrefilterReject(watcher FcapSnapshotProvider, catalog *RtbCatalog, targeting RtbTargetingInput) NoBidReason {
	if watcher != nil && watcher.RTBEmergencyBreaker() {
		return NoBidBreakerOpen
	}
	if catalog == nil || catalog.registry == nil {
		return NoBidNone
	}
	if targeting.GeoHash == 0 {
		return NoBidNone
	}
	shard := catalog.registry.LoadShard(targeting.GeoHash)
	if shard == nil || shard.Count == 0 {
		return NoBidNoCandidates
	}
	return NoBidNone
}

func rtbPrebidIVTReject(enabled bool, geo GeoAnonLookup, evt *domain.Event) NoBidReason {
	if !enabled || evt == nil || geo == nil || evt.IP == "" {
		return NoBidNone
	}
	anon, err := geo.IsAnonymous(evt.IP)
	if err == nil && anon {
		return NoBidPrebidIVT
	}
	return NoBidNone
}

type (
	SchainNode                   = openrtb.SupplyChainNode
	SchainNodes                  = openrtb.SupplyChainNodes
	SupplyChainAllowlistSnapshot = openrtb.SupplyChainAllowlistSnapshot
)

func ValidateSchainNodes(nodes SchainNodes, allow *SupplyChainAllowlistSnapshot) bool {
	return openrtb.ValidateSupplyChainNodes(nodes, allow)
}

func schainAllowKey(asi, sid []byte) string {
	return openrtb.SchainAllowKey(asi, sid)
}

func BuildSupplyChainAllowlistFromSellers(domains []string, sellerIDs []string) *SupplyChainAllowlistSnapshot {
	if len(domains) == 0 || len(domains) != len(sellerIDs) {
		return &SupplyChainAllowlistSnapshot{Allowed: make(map[string]struct{})}
	}
	allowed := make(map[string]struct{}, len(domains))
	for i := range domains {
		key := schainAllowKey([]byte(domains[i]), []byte(sellerIDs[i]))
		if key != "" {
			allowed[key] = struct{}{}
		}
	}
	return &SupplyChainAllowlistSnapshot{Allowed: allowed}
}

func LoadSupplyChainAllowlist(ctx context.Context, q *db.Queries) (*SupplyChainAllowlistSnapshot, error) {
	if q == nil {
		return &SupplyChainAllowlistSnapshot{Allowed: make(map[string]struct{})}, nil
	}
	rows, err := q.ListSellers(ctx)
	if err != nil {
		return nil, err
	}
	domains := make([]string, 0, len(rows))
	sellerIDs := make([]string, 0, len(rows))
	for _, row := range rows {
		domains = append(domains, row.Domain)
		sellerIDs = append(sellerIDs, row.SellerID)
	}
	return BuildSupplyChainAllowlistFromSellers(domains, sellerIDs), nil
}

const rtbShadowDiffBuckets = 24

type RtbShadowDiffSnapshotDTO struct {
	Window            string  `json:"window"`
	Source            string  `json:"source"`
	ShadowEvals       uint64  `json:"shadow_evals"`
	ShadowWinnerMatch uint64  `json:"shadow_winner_match"`
	ShadowMismatch    uint64  `json:"shadow_winner_mismatch"`
	ShadowNoBid       uint64  `json:"shadow_no_bid"`
	LiveWouldAccept   uint64  `json:"live_would_accept"`
	LiveWouldReject   uint64  `json:"live_would_reject"`
	ParityMatch       uint64  `json:"parity_match"`
	ParityRate        float64 `json:"parity_rate"`
	MismatchRate      float64 `json:"mismatch_rate"`
}

type rtbShadowDiffBucket struct {
	shadowEvals       atomic.Uint64
	shadowWinnerMatch atomic.Uint64
	shadowMismatch    atomic.Uint64
	shadowNoBid       atomic.Uint64
	liveWouldAccept   atomic.Uint64
	liveWouldReject   atomic.Uint64
	parityMatch       atomic.Uint64
}

var rtbShadowDiffRing [rtbShadowDiffBuckets]rtbShadowDiffBucket

func rtbShadowDiffBucketIdx(now time.Time) int {
	return now.UTC().Hour() % rtbShadowDiffBuckets
}

func recordRtbShadowDiff(catalog *RtbCatalog, evt *domain.Event, res AuctionResult, reason NoBidReason) {
	if catalog == nil || evt == nil || evt.CampaignID == uuid.Nil {
		return
	}
	b := &rtbShadowDiffRing[rtbShadowDiffBucketIdx(time.Now())]
	b.shadowEvals.Add(1)

	if !reason.OK() {
		b.shadowNoBid.Add(1)
		b.liveWouldReject.Add(1)
		b.parityMatch.Add(1)
		return
	}

	b.liveWouldAccept.Add(1)
	shadowWinner, ok := catalog.UUIDForWinner(res.CampaignID)
	if !ok || shadowWinner != evt.CampaignID {
		b.shadowMismatch.Add(1)
		return
	}
	b.shadowWinnerMatch.Add(1)
	b.parityMatch.Add(1)
}

func RtbShadowDiffForWindow(window time.Duration) RtbShadowDiffSnapshotDTO {
	if window <= 0 {
		window = time.Hour
	}
	hours := int(window.Hours())
	if hours < 1 {
		hours = 1
	}
	if hours > rtbShadowDiffBuckets {
		hours = rtbShadowDiffBuckets
	}

	now := time.Now().UTC()
	var snap RtbShadowDiffSnapshotDTO
	snap.Window = window.String()
	snap.Source = "memory"

	for i := range hours {
		idx := (now.Hour() - i + rtbShadowDiffBuckets) % rtbShadowDiffBuckets
		b := &rtbShadowDiffRing[idx]
		snap.ShadowEvals += b.shadowEvals.Load()
		snap.ShadowWinnerMatch += b.shadowWinnerMatch.Load()
		snap.ShadowMismatch += b.shadowMismatch.Load()
		snap.ShadowNoBid += b.shadowNoBid.Load()
		snap.LiveWouldAccept += b.liveWouldAccept.Load()
		snap.LiveWouldReject += b.liveWouldReject.Load()
		snap.ParityMatch += b.parityMatch.Load()
	}

	if snap.ShadowEvals > 0 {
		snap.ParityRate = float64(snap.ParityMatch) / float64(snap.ShadowEvals)
		snap.MismatchRate = float64(snap.ShadowMismatch) / float64(snap.ShadowEvals)
	}
	return snap
}

func ResetRtbShadowDiffBuckets() {
	for i := range rtbShadowDiffRing {
		b := &rtbShadowDiffRing[i]
		b.shadowEvals.Store(0)
		b.shadowWinnerMatch.Store(0)
		b.shadowMismatch.Store(0)
		b.shadowNoBid.Store(0)
		b.liveWouldAccept.Store(0)
		b.liveWouldReject.Store(0)
		b.parityMatch.Store(0)
	}
}

const rtbShadowPriceSampleMask uint64 = 127

type preboundRtbShadowMetrics struct {
	winnerMismatch prometheus.Counter
	noBid          map[NoBidReason]prometheus.Counter
	priceDelta     prometheus.Observer
}

var (
	rtbShadowMetrics     preboundRtbShadowMetrics
	rtbShadowMetricsInit atomic.Bool
	rtbShadowPriceSeq    atomic.Uint64
)

func bindRtbShadowMetrics() {
	if rtbShadowMetricsInit.Swap(true) {
		return
	}
	noBid := make(map[NoBidReason]prometheus.Counter, 8)
	for reason := NoBidInvalidRequest; reason <= NoBidDailyCapExceeded; reason++ {
		noBid[reason] = metrics.RtbShadowNoBidTotal.WithLabelValues(reason.String())
	}
	rtbShadowMetrics = preboundRtbShadowMetrics{
		winnerMismatch: metrics.RtbShadowWinnerMismatchTotal,
		noBid:          noBid,
		priceDelta:     metrics.RtbShadowPriceDeltaMicro,
	}
}

func init() {
	bindRtbShadowMetrics()
}

func RecordRtbShadowAuction(
	catalog *RtbCatalog,
	evt *domain.Event,
	res AuctionResult,
	reason NoBidReason,
	payloadBidMicro int64,
) {
	if catalog == nil || evt == nil {
		return
	}
	if evt.CampaignID == uuid.Nil {
		return
	}
	if !reason.OK() {
		if counter, ok := rtbShadowMetrics.noBid[reason]; ok {
			counter.Inc()
		}
	} else {
		shadowWinner, ok := catalog.UUIDForWinner(res.CampaignID)
		if !ok || shadowWinner != evt.CampaignID {
			rtbShadowMetrics.winnerMismatch.Inc()
		}
	}
	recordRtbShadowDiff(catalog, evt, res, reason)
	if payloadBidMicro <= 0 {
		return
	}
	seq := rtbShadowPriceSeq.Add(1)
	if seq&rtbShadowPriceSampleMask != 0 {
		return
	}
	delta := res.Price - payloadBidMicro
	if delta < 0 {
		delta = -delta
	}
	rtbShadowMetrics.priceDelta.Observe(float64(delta))
}

const (
	rtbDeviceMaskAll    uint8  = 7
	rtbDaySeconds       int64  = 86400
	defaultHybridMaxRPS        = 5000
	defaultCategoryMask uint64 = 1
)

func BuildCampaignMetaList(campaigns []*domain.Campaign, cfg *config.Config) []*CampaignMeta {
	if len(campaigns) == 0 || cfg == nil {
		return nil
	}
	bidDefault := defaultBidMicro(cfg)
	out := make([]*CampaignMeta, 0, len(campaigns))
	for _, camp := range campaigns {
		if camp == nil {
			continue
		}
		total := camp.BudgetLimit
		if total < 0 {
			total = 0
		}
		out = append(out, &CampaignMeta{
			ID:                camp.ID,
			BidMicro:          bidDefault,
			CTR:               1.0,
			RemainingBudget:   remainingBudgetMicro(camp),
			TotalBudget:       total,
			PeakTrafficFactor: 1.0,
		})
	}
	return out
}

func defaultBidMicro(cfg *config.Config) int64 {
	bidMicro := cfg.ClickAmount
	if bidMicro <= 0 {
		bidMicro = cfg.ImpressionAmount
	}
	if bidMicro <= 0 {
		bidMicro = 1
	}
	return bidMicro
}

func campaignMetaByID(metas []*CampaignMeta) map[uuid.UUID]*CampaignMeta {
	if len(metas) == 0 {
		return nil
	}
	out := make(map[uuid.UUID]*CampaignMeta, len(metas))
	for _, meta := range metas {
		if meta != nil {
			out[meta.ID] = meta
		}
	}
	return out
}

func buildCustomerBudgetPools(campaigns []*domain.Campaign) map[uuid.UUID]int64 {
	if len(campaigns) == 0 {
		return nil
	}
	out := make(map[uuid.UUID]int64)
	for _, camp := range campaigns {
		if camp == nil || camp.CustomerID == uuid.Nil {
			continue
		}
		out[camp.CustomerID] += remainingBudgetMicro(camp)
	}
	return out
}

func BuildRtbInputsFromRegistry(
	registry CampaignSource,
	cfg *config.Config,
	metaByID map[uuid.UUID]*CampaignMeta,
	customerPools map[uuid.UUID]int64,
	hybrid CampaignWeighter,
	boosts *FraudBoostSnapshot,
) map[uuid.UUID]RtbCampaignInput {
	if registry == nil || cfg == nil {
		return nil
	}
	campaigns := registry.ActiveCampaigns()
	if len(campaigns) == 0 {
		return nil
	}
	out := make(map[uuid.UUID]RtbCampaignInput, len(campaigns))
	for _, camp := range campaigns {
		if camp == nil {
			continue
		}
		out[camp.ID] = rtbInputForCampaign(camp, cfg, metaByID[camp.ID], customerPools[camp.CustomerID], hybrid, boosts)
	}
	return out
}

func rtbInputForCampaign(
	camp *domain.Campaign,
	cfg *config.Config,
	meta *CampaignMeta,
	customerBudget int64,
	hybrid CampaignWeighter,
	boosts *FraudBoostSnapshot,
) RtbCampaignInput {
	geo := firstTargetCountryGeo(camp)
	pacing := PacingOpenFromManagement(camp.Status == domain.CampaignStatusActive)
	customerID := CustomerIDFromCustomerUUID(camp.CustomerID)
	dailyMicro := camp.DailyBudgetMicro
	if dailyMicro <= 0 {
		dailyMicro = camp.DailyBudget
	}
	weight := uint32(1)
	if hybrid != nil {
		weight = hybrid.WeightFor(camp.ID)
	}
	boostPPM := uint32(CTRPPMUnit)
	if boosts != nil {
		if b, ok := boosts.Boosts[camp.ID]; ok {
			boostPPM = BoostPPMFromUint8(b)
		}
	}
	if meta != nil {
		inp := RtbCampaignInputFromHybrid(
			meta,
			geo,
			rtbDeviceMaskAll,
			defaultCategoryMask,
			weight,
			pacing,
			customerID,
			customerBudget,
			dailyMicro,
		)
		inp.ReserveMicro = camp.ReserveMicro
		inp.BoostPPM = boostPPM
		return inp
	}
	return RtbCampaignInput{
		BidMicro:         defaultBidMicro(cfg),
		CTRPPM:           CTRPPMUnit,
		ReserveMicro:     camp.ReserveMicro,
		DeviceMask:       rtbDeviceMaskAll,
		CategoryMask:     defaultCategoryMask,
		GeoHash:          geo,
		Weight:           weight,
		BoostPPM:         boostPPM,
		PacingOpen:       pacing,
		CustomerID:       customerID,
		CustomerBudget:   customerBudget,
		DailyBudgetMicro: dailyMicro,
	}
}

func firstTargetCountryGeo(camp *domain.Campaign) uint32 {
	if camp == nil || len(camp.TargetCountries) == 0 {
		return 0
	}
	countries := make([]string, 0, len(camp.TargetCountries))
	for c := range camp.TargetCountries {
		countries = append(countries, c)
	}
	sort.Strings(countries)
	return GeoHashFromCountry(countries[0])
}

func BudgetAuthorityFromConfig(cfg *config.Config) BudgetAuthority {
	return BudgetAuthorityFromSettings(cfg, "")
}

func utcSecondsElapsed() int64 {
	now := time.Now().UTC()
	return int64(now.Hour()*3600 + now.Minute()*60 + now.Second())
}

func SyncRtbCatalog(
	ctx context.Context,
	registry CampaignSource,
	catalog *RtbCatalog,
	cfg *config.Config,
	hybrid CampaignWeighter,
	budgetSync RtbBudgetSync,
	watcher FcapSnapshotProvider,
) {
	if registry == nil || catalog == nil || cfg == nil {
		return
	}
	campaigns := registry.ActiveCampaigns()
	metas := BuildCampaignMetaList(campaigns, cfg)
	metaByID := campaignMetaByID(metas)
	if hybrid != nil {
		hybrid.UpdateCampaigns(metas, utcSecondsElapsed(), rtbDaySeconds)
	}
	customerPools := buildCustomerBudgetPools(campaigns)
	var boosts *FraudBoostSnapshot
	if watcher != nil {
		boosts = watcher.FraudBoosts()
	}
	inputs := BuildRtbInputsFromRegistry(registry, cfg, metaByID, customerPools, hybrid, boosts)
	rows := BuildRtbCatalogRowsFromHybrid(campaigns, metaByID, inputs)
	catalog.SyncCampaignRows(campaigns, rows)
	if watcher != nil {
		catalog.registry.SetFcapSnapshot(watcher.GetFcapRtbSnapshot())
	}
	SyncRTBBudgetState(ctx, catalog.Registry().Store(), campaigns, customerPools, budgetSync)
}

func StartRtbCatalogSync(
	ctx context.Context,
	registry CampaignSource,
	catalog *RtbCatalog,
	cfg *config.Config,
	hybrid CampaignWeighter,
	budgetSync RtbBudgetSync,
	watcher FcapSnapshotProvider,
	interval time.Duration,
) {
	if registry == nil || catalog == nil || cfg == nil || interval <= 0 {
		return
	}
	syncOnce := func() {
		syncCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		SyncRtbCatalog(syncCtx, registry, catalog, cfg, hybrid, budgetSync, watcher)
		cancel()
	}
	syncOnce()
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				syncOnce()
			}
		}
	}()
}

func HybridMaxRPSFromConfig(cfg *config.Config) int {
	if cfg == nil || cfg.RtbHybridMaxRpsPerNode <= 0 {
		return defaultHybridMaxRPS
	}
	return cfg.RtbHybridMaxRpsPerNode
}

const (
	RtbModeOff uint8 = iota
	RtbModeShadow
	RtbModeLive
)

func RtbModeFromConfig(cfg *config.Config) uint8 {
	if cfg == nil {
		return RtbModeOff
	}
	switch config.ParseRtbMode(cfg.RtbMode) {
	case config.RtbModeShadow:
		return RtbModeShadow
	case config.RtbModeLive:
		return RtbModeLive
	default:
		return RtbModeOff
	}
}
