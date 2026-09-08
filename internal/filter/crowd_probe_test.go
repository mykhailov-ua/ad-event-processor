package filter

import (
	"context"
	"testing"
	"time"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/filter/netintel"
	"ad-event-processor/internal/metrics"
	"ad-event-processor/pkg/crowdprobe"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type crowdProbeCampRegistry struct {
	camp *domain.Campaign
}

func (r *crowdProbeCampRegistry) Exists(_ uuid.UUID) bool { return r.camp != nil }
func (r *crowdProbeCampRegistry) Add(_ uuid.UUID, _ uuid.UUID, _ *uuid.UUID, _ string, _ domain.PacingMode, _ int64, _ string, _ int32, _ int32, _ []string) {
}
func (r *crowdProbeCampRegistry) GetCustomerID(_ uuid.UUID) (uuid.UUID, bool) { return uuid.Nil, false }
func (r *crowdProbeCampRegistry) GetCampaign(_ uuid.UUID) (*domain.Campaign, bool) {
	if r.camp == nil {
		return nil, false
	}
	return r.camp, true
}
func (r *crowdProbeCampRegistry) Sync(_ context.Context) (int, error)          { return 0, nil }
func (r *crowdProbeCampRegistry) StartSync(_ context.Context, _ time.Duration) {}
func (r *crowdProbeCampRegistry) Wait(_ context.Context) error                 { return nil }

type stubASNLookup struct {
	asn uint32
	ok  bool
}

func (s stubASNLookup) LookupASN(string) (uint32, bool) {
	return s.asn, s.ok
}

func TestCrowdProbeFilter_holdoutSOPFlagsBehavior(t *testing.T) {
	before := testutil.ToFloat64(metrics.CrowdProbeSignalTotal)
	campID := uuid.New()
	reg := &crowdProbeCampRegistry{
		camp: &domain.Campaign{
			ID:                 campID,
			SafePageEnabled:    true,
			AttestationEnabled: true,
		},
	}
	f := NewCrowdProbeFilter(reg)
	f.SetEnabled(true)
	f.SetPriorReader(crowdProbePriorStub{tier: 3, prior: 500})
	f.SetASNLookup(stubASNLookup{asn: 12345, ok: true})

	evt := &domain.Event{
		CampaignID:   campID,
		Type:         "conversion",
		UA:           "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X)",
		IP:           "198.51.100.10",
		TelemetrySet: 1,
		AntifraudSet: 1,
		AntifraudSnapshot: domain.AntifraudSnapshot{
			DwellMs:       1800,
			FooterReachMs: 1200,
			ScrollCVMilli: 18,
		},
		TelemetryEvents: sopCorpusTelemetry(),
	}
	acc := AttachFraudAccumulator(evt)
	require.NoError(t, f.Check(context.Background(), evt))
	assert.True(t, acc.Has(FraudReasonCrowdProbeBehavior))
	assert.Equal(t, uint8(1), evt.ProbeFeaturesSet)
	assert.GreaterOrEqual(t, int(evt.ProbeBehaviorScore), crowdprobe.BehaviorSignalThreshold)
	assert.GreaterOrEqual(t, testutil.ToFloat64(metrics.CrowdProbeSignalTotal), before+1)
}

func TestCrowdProbeFilter_holdoutOrganicClean(t *testing.T) {
	campID := uuid.New()
	reg := &crowdProbeCampRegistry{
		camp: &domain.Campaign{
			ID:                 campID,
			SafePageEnabled:    true,
			AttestationEnabled: true,
		},
	}
	f := NewCrowdProbeFilter(reg)
	f.SetEnabled(true)

	evt := &domain.Event{
		CampaignID:   campID,
		Type:         "conversion",
		UA:           "Mozilla/5.0 (Windows NT 10.0; Win64; x64)",
		TelemetrySet: 1,
		AntifraudSet: 1,
		AntifraudSnapshot: domain.AntifraudSnapshot{
			DwellMs:       9000,
			FooterReachMs: 7000,
			ScrollCVMilli: 220,
		},
		TelemetryEvents: organicCorpusTelemetry(),
	}
	acc := AttachFraudAccumulator(evt)
	require.NoError(t, f.Check(context.Background(), evt))
	assert.False(t, acc.Has(FraudReasonCrowdProbeBehavior))
	assert.False(t, acc.Has(FraudReasonCrowdProbeTiming))
}

func sopCorpusTelemetry() []domain.BehaviorTelemetryEvent {
	out := make([]domain.BehaviorTelemetryEvent, 20)
	for i := range out {
		t := "mousemove"
		switch i % 4 {
		case 0:
			t = "pointerdown"
		case 1:
			t = "scroll"
		case 2:
			t = "click"
		case 3:
			t = "visibilitychange"
		}
		out[i] = domain.BehaviorTelemetryEvent{
			T: t, TS: int64(i * 80), X: i * 12, Y: i * 12,
			FX: float32(i * 12), FY: float32(i * 12), Trusted: 1,
		}
	}
	return out
}

func organicCorpusTelemetry() []domain.BehaviorTelemetryEvent {
	types := []string{"mousemove", "pointerdown", "scroll", "keydown", "click", "mousemove", "visibilitychange"}
	out := make([]domain.BehaviorTelemetryEvent, 24)
	for i := range out {
		out[i] = domain.BehaviorTelemetryEvent{
			T: types[i%len(types)], TS: int64(i*140 + (i%5)*17),
			X: 100 + i*7 + (i%3)*11, Y: 200 + i*5 + (i%4)*7,
			FX: float32(100+i*7+(i%3)*11) + 1, FY: float32(200+i*5+(i%4)*7) + 1, Trusted: 1,
		}
	}
	return out
}

func TestCrowdProbeFilter_holdoutASNMobileTierSignal(t *testing.T) {
	before := testutil.ToFloat64(metrics.CrowdProbeASNSignalTotal)
	campID := uuid.New()
	reg := &crowdProbeCampRegistry{
		camp: &domain.Campaign{
			ID:                 campID,
			SafePageEnabled:    true,
			AttestationEnabled: true,
		},
	}
	mobileTier := netintel.NewMobileTierTable()
	f := NewCrowdProbeFilter(reg)
	f.SetEnabled(true)
	f.SetPriorReader(crowdProbePriorStub{tier: 3, prior: 500})
	f.SetASNLookup(stubASNLookup{asn: 310410, ok: true})
	f.SetMobileTierTable(mobileTier)
	f.SetASNMobileRisk(netintel.DefaultMobileProbeRiskWeights(), 3, crowdprobe.BehaviorSignalThreshold)

	evt := &domain.Event{
		CampaignID:   campID,
		Type:         "conversion",
		UA:           "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X)",
		IP:           "198.51.100.10",
		TelemetrySet: 1,
		AntifraudSet: 1,
		AntifraudSnapshot: domain.AntifraudSnapshot{
			DwellMs:       1800,
			FooterReachMs: 1200,
			ScrollCVMilli: 18,
		},
		TelemetryEvents: sopCorpusTelemetry(),
	}
	acc := AttachFraudAccumulator(evt)
	require.NoError(t, f.Check(context.Background(), evt))
	assert.True(t, acc.Has(FraudReasonCrowdProbeASN))
	assert.GreaterOrEqual(t, testutil.ToFloat64(metrics.CrowdProbeASNSignalTotal), before+1)
}
