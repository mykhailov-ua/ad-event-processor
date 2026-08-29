package stream

import "ad-event-processor/internal/domain"

type UDPNodeWeight = domain.UDPNodeWeight

type ProcessorUDPWeightsSource interface {
	NodeWeights() []UDPNodeWeight
}
