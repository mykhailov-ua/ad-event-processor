package stream

import "ad-event-processor/internal/domain"

type UDPNodeWeight = domain.UDPNodeWeight

// ProcessorUDPWeightsSource supplies per-node ingest weights from tracker UDP gossip.
type ProcessorUDPWeightsSource interface {
	NodeWeights() []UDPNodeWeight
}
