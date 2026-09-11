package domain

import "strings"

type FlowRoutingMode string

const (
	FlowRoutingModeWeighted             FlowRoutingMode = "weighted"
	FlowRoutingModeWaterfall            FlowRoutingMode = "waterfall"
	FlowRoutingModeWaterfallThenLanding FlowRoutingMode = "waterfall_then_landing"
)

func NormalizeFlowRoutingMode(raw string) FlowRoutingMode {
	switch strings.TrimSpace(strings.ToLower(raw)) {
	case "waterfall":
		return FlowRoutingModeWaterfall
	case "waterfall_then_landing":
		return FlowRoutingModeWaterfallThenLanding
	default:
		return FlowRoutingModeWeighted
	}
}
