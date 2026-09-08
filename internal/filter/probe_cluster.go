package filter

import (
	"context"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/metrics"
	"ad-event-processor/pkg/probecluster"
)

type ProbeClusterFilter struct {
	registry domain.CampaignRegistry
	enabled  bool
	secret   []byte
	store    *ProbeClusterStore
	policy   probecluster.Policy
}

func NewProbeClusterFilter(registry domain.CampaignRegistry) *ProbeClusterFilter {
	return &ProbeClusterFilter{
		registry: registry,
		policy:   probecluster.DefaultPolicy(),
	}
}

func (f *ProbeClusterFilter) SetEnabled(enabled bool) {
	if f != nil {
		f.enabled = enabled
	}
}

func (f *ProbeClusterFilter) SetSecret(secret []byte) {
	if f != nil {
		f.secret = secret
	}
}

func (f *ProbeClusterFilter) SetStore(store *ProbeClusterStore) {
	if f != nil {
		f.store = store
	}
}

func (f *ProbeClusterFilter) SetPolicy(policy probecluster.Policy) {
	if f != nil {
		f.policy = policy
	}
}

func (f *ProbeClusterFilter) Check(ctx context.Context, evt *domain.Event) error {
	if f == nil || evt == nil || !f.enabled || f.store == nil || len(f.secret) == 0 {
		return nil
	}
	if evt.Type != "click" && evt.Type != "conversion" {
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
	snap := domain.AntifraudSnapshot{}
	if evt.AntifraudSet != 0 {
		snap = evt.AntifraudSnapshot
	}
	tuple := probecluster.TupleFromEvent(evt, snap)
	if !tupleHasSignal(tuple) {
		return nil
	}
	clusterID := probecluster.ClusterID(f.secret, tuple)
	evt.ProbeClusterID = clusterID
	evt.ProbeClusterSet = 1
	sessionID := evt.ClickID
	if sessionID == "" {
		sessionID = evt.UserID
	}
	if sessionID == "" {
		return nil
	}
	probeScore := evt.ProbeBehaviorScore
	if probeScore == 0 && evt.ProbeFeaturesSet == 0 {
		probeScore = 1
	}
	verifyIncr := evt.Type == "conversion" && evt.AntifraudSet != 0
	state, err := f.store.Observe(ctx, ProbeClusterObserveInput{
		ClusterID:  clusterID,
		SessionID:  sessionID,
		CampaignID: evt.CampaignID,
		ProbeScore: probeScore,
		VerifyIncr: verifyIncr,
		JA3:        evt.TLSJA3,
		JA4:        evt.TLSJA4,
		TCPSig:     evt.TCPSig,
		TCPSigSet:  evt.TCPSigSet,
		WebGLHex:   probecluster.Hash16Hex(snap.WebGLHash),
	})
	if err != nil {
		return nil
	}
	if probecluster.ShouldRouteSandbox(state, f.policy) {
		evt.ProbeClusterRoute = 1
		metrics.ProbeClusterRouteTotal.Inc()
	}
	return nil
}

func tupleHasSignal(t probecluster.Tuple) bool {
	if t.TLSJA3 != "" || t.TLSJA4 != "" {
		return true
	}
	if t.TCPSigSet != 0 {
		return true
	}
	if t.CanvasHash[0] != 0 || t.AudioHash[0] != 0 || t.WebGLHash[0] != 0 {
		return true
	}
	return false
}
