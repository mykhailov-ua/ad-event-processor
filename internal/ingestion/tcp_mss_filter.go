package ingestion

import (
	"context"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/metrics"
)

// TCPMSSFilter emits an L2 fraud signal when edge SYN MSS (high-byte encoding) is below threshold.
type TCPMSSFilter struct {
	minMSS uint8
}

func NewTCPMSSFilter(minMSS uint8) *TCPMSSFilter {
	if minMSS == 0 {
		minMSS = 2
	}
	return &TCPMSSFilter{minMSS: minMSS}
}

func (f *TCPMSSFilter) Check(ctx context.Context, evt *domain.Event) error {
	if f == nil || evt == nil || evt.TCPMSSSet == 0 {
		return nil
	}
	if evt.TCPMSS >= f.minMSS {
		return nil
	}
	metrics.TCPMSSAnomalyTotal.WithLabelValues("low_mss").Inc()
	addFraudSignal(evt, FraudReasonTCPMSSAnomaly)
	return nil
}
