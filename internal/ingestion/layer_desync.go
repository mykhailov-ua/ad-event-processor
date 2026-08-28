package ingestion

import (
	"math/bits"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/metrics"
)

const (
	fraudDesyncLayerTCPOS       uint8 = 1 << 0
	fraudDesyncLayerTLSJA4      uint8 = 1 << 1
	fraudDesyncLayerClientHints uint8 = 1 << 2
	fraudDesyncLayerSecFetch    uint8 = 1 << 3
	fraudDesyncLayerH2          uint8 = 1 << 4
)

func fraudDesyncLayerBit(id FraudReasonID) uint8 {
	switch id {
	case FraudReasonTCPSynOSMismatch:
		return fraudDesyncLayerTCPOS
	case FraudReasonTLSJA4Mismatch:
		return fraudDesyncLayerTLSJA4
	case FraudReasonClientHintsMismatch:
		return fraudDesyncLayerClientHints
	case FraudReasonSecFetchAnomaly:
		return fraudDesyncLayerSecFetch
	case FraudReasonH2SettingsMismatch, FraudReasonH2PseudoOrder, FraudReasonH2DowngradeArtifact:
		return fraudDesyncLayerH2
	default:
		return 0
	}
}

func (a *fraudAccumulator) layerDesyncCount() uint8 {
	if a == nil || a.count == 0 {
		return 0
	}
	var mask uint8
	for i := uint8(0); i < a.count; i++ {
		mask |= fraudDesyncLayerBit(a.signals[i])
	}
	return uint8(bits.OnesCount8(mask))
}

var fraudStreamLayerDesyncLabels = [6]string{"0", "1", "2", "3", "4", "5"}

func observeFraudStreamLayerDesync(count uint8) {
	if count > 5 {
		count = 5
	}
	metrics.FraudStreamLayerDesyncTotal.WithLabelValues(fraudStreamLayerDesyncLabels[count]).Inc()
}

func isFraudStreamLayerDesyncTelemetry(evt *domain.Event) bool {
	if evt == nil {
		return false
	}
	return evt.FraudReason != "" || evt.FraudScore > 0
}
