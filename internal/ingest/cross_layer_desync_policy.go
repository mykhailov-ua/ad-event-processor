package ingest

import (
	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/metrics"
)

type crossLayerDesyncPolicyResult struct {
	Fired         bool
	ForceSafePage bool
	Block         bool
}

func evalCrossLayerDesyncClickPolicy(registry domain.CampaignRegistry, evt *domain.Event) crossLayerDesyncPolicyResult {
	var res crossLayerDesyncPolicyResult
	if evt == nil || registry == nil {
		return res
	}
	camp, ok := registry.GetCampaign(evt.CampaignID)
	if !ok || camp == nil {
		return res
	}
	action := camp.CrossLayerDesyncAction
	if action == domain.CrossLayerDesyncOff || action == domain.CrossLayerDesyncBoost {
		return res
	}
	threshold := domain.NormalizeCrossLayerDesyncThreshold(camp.CrossLayerDesyncThreshold)
	if evt.LayerDesyncCount < threshold {
		return res
	}
	res.Fired = true
	metrics.CrossLayerDesyncFiredTotal.Inc()
	switch action {
	case domain.CrossLayerDesyncSafePage:
		res.ForceSafePage = true
	case domain.CrossLayerDesyncBlock:
		res.Block = true
	}
	return res
}
