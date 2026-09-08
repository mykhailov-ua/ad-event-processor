package filter

import (
	"context"
	"strconv"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/filter/netintel"
	"ad-event-processor/internal/metrics"
	"ad-event-processor/pkg/crowdprobe"

	"github.com/redis/go-redis/v9"
)

type CrowdProbePriorReader interface {
	LookupPrior(ctx context.Context, asn uint32) (tier uint8, clusterPriorMilli uint16, ok bool)
}

type CrowdProbeFilter struct {
	registry         domain.CampaignRegistry
	enabled          bool
	asnMobileEnabled bool
	prior            CrowdProbePriorReader
	asnLookup        ASNLookup
	mobileTier       *netintel.MobileTierTable
	proxyRing        *netintel.ResidentialProxyRing
	riskWeights      netintel.MobileProbeRiskWeights
	minASNTier       uint8
	minASNScore      uint8
}

func NewCrowdProbeFilter(registry domain.CampaignRegistry) *CrowdProbeFilter {
	return &CrowdProbeFilter{registry: registry}
}

func (f *CrowdProbeFilter) SetEnabled(enabled bool) {
	if f != nil {
		f.enabled = enabled
	}
}

func (f *CrowdProbeFilter) SetPriorReader(reader CrowdProbePriorReader) {
	if f != nil {
		f.prior = reader
	}
}

func (f *CrowdProbeFilter) SetASNLookup(lookup ASNLookup) {
	if f != nil {
		f.asnLookup = lookup
	}
}

func (f *CrowdProbeFilter) SetMobileTierTable(table *netintel.MobileTierTable) {
	if f != nil {
		f.mobileTier = table
	}
}

func (f *CrowdProbeFilter) SetResidentialProxyRing(ring *netintel.ResidentialProxyRing) {
	if f != nil {
		f.proxyRing = ring
	}
}

func (f *CrowdProbeFilter) SetASNMobileRisk(weights netintel.MobileProbeRiskWeights, minTier, minScore uint8) {
	if f == nil {
		return
	}
	f.riskWeights = weights
	f.minASNTier = minTier
	f.minASNScore = minScore
	f.asnMobileEnabled = minTier > 0 || minScore > 0
}

const crowdProbePriorKeyPrefix = "probe:prior:"

func (f *CrowdProbeFilter) Check(ctx context.Context, evt *domain.Event) error {
	if f == nil || evt == nil || !f.enabled {
		return nil
	}
	if evt.Type != "conversion" && evt.Type != "click" {
		return nil
	}
	if ScanUAFamily(evt.UA) == UAFamilyUnknown {
		return nil
	}
	if f.registry == nil {
		return nil
	}
	camp, ok := f.registry.GetCampaign(evt.CampaignID)
	if !ok || camp == nil {
		return nil
	}
	if !campaignRequiresCrowdProbe(camp) {
		return nil
	}
	if evt.TelemetrySet == 0 && evt.AntifraudSet == 0 {
		return nil
	}
	snap := domain.AntifraudSnapshot{}
	if evt.AntifraudSet != 0 {
		snap = evt.AntifraudSnapshot
	}
	features := crowdprobe.ComputeSessionFeatures(evt.TelemetryEvents, snap)
	tier := uint8(0)
	priorMilli := uint16(0)
	if f.prior != nil && f.asnLookup != nil && evt.IP != "" {
		asn, asnOK := f.asnLookup.LookupASN(evt.IP)
		if asnOK {
			tier, priorMilli, _ = f.prior.LookupPrior(ctx, asn)
		}
	}
	result := crowdprobe.Score(features, tier, priorMilli)
	crowdprobe.ApplyEventFields(evt, features, result)
	if result.BehaviorHit {
		metrics.CrowdProbeSignalTotal.Inc()
		AddFraudSignal(evt, FraudReasonCrowdProbeBehavior)
	}
	if result.TimingHit {
		metrics.CrowdProbeSignalTotal.Inc()
		AddFraudSignal(evt, FraudReasonCrowdProbeTiming)
	}
	f.maybeFlagCrowdProbeASN(evt, result.BehaviorScore, priorMilli)
	return nil
}

func (f *CrowdProbeFilter) maybeFlagCrowdProbeASN(evt *domain.Event, probeScore uint8, clusterPriorMilli uint16) {
	if f == nil || evt == nil || !f.asnMobileEnabled || f.mobileTier == nil || f.asnLookup == nil || evt.IP == "" {
		return
	}
	asn, asnOK := f.asnLookup.LookupASN(evt.IP)
	if !asnOK || asn == 0 {
		return
	}
	mobileTier := f.mobileTier.MobileTier(asn)
	minTier := f.minASNTier
	if minTier == 0 {
		minTier = 3
	}
	minScore := f.minASNScore
	if minScore == 0 {
		minScore = crowdprobe.BehaviorSignalThreshold
	}
	if mobileTier < minTier || probeScore < minScore {
		return
	}
	clusterHistory := float64(clusterPriorMilli) / 1000
	if f.proxyRing != nil {
		campaignHash := CRC32Castagnoli(&evt.CampaignID)
		if _, proxySignal := f.proxyRing.Peek(campaignHash); proxySignal {
			clusterHistory = 1
		}
	}
	weights := f.riskWeights
	if weights.Alpha == 0 && weights.Beta == 0 && weights.Gamma == 0 {
		weights = netintel.DefaultMobileProbeRiskWeights()
	}
	_ = netintel.ComputeMobileProbeRisk(weights, mobileTier, probeScore, clusterHistory)
	metrics.CrowdProbeASNSignalTotal.Inc()
	AddFraudSignal(evt, FraudReasonCrowdProbeASN)
}

func campaignRequiresCrowdProbe(camp *domain.Campaign) bool {
	return camp.SafePageEnabled && camp.AttestationEnabled
}

type crowdProbeRedisPrior struct {
	client redis.UniversalClient
}

func NewCrowdProbeRedisPrior(client redis.UniversalClient) CrowdProbePriorReader {
	if client == nil {
		return nil
	}
	return &crowdProbeRedisPrior{client: client}
}

func (r *crowdProbeRedisPrior) LookupPrior(ctx context.Context, asn uint32) (uint8, uint16, bool) {
	if r == nil || r.client == nil || asn == 0 {
		return 0, 0, false
	}
	key := crowdProbePriorKey(asn)
	b, err := r.client.Get(ctx, key).Bytes()
	if err != nil || len(b) < 3 {
		return 0, 0, false
	}
	tier := b[0]
	prior := uint16(b[1])<<8 | uint16(b[2])
	return tier, prior, true
}

func crowdProbePriorKey(asn uint32) string {
	buf := make([]byte, 0, len(crowdProbePriorKeyPrefix)+10)
	buf = append(buf, crowdProbePriorKeyPrefix...)
	buf = strconv.AppendUint(buf, uint64(asn), 10)
	return string(buf)
}

type crowdProbePriorStub struct {
	tier  uint8
	prior uint16
}

func (s crowdProbePriorStub) LookupPrior(context.Context, uint32) (uint8, uint16, bool) {
	return s.tier, s.prior, true
}
