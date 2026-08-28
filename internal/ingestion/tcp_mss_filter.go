package ingestion

import (
	"context"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/metrics"
)

type TCPMSSFilter struct {
	minMSS           uint8
	tunnelEnabled    bool
	tunnelThreshold  uint16
	mobileCarrierASN *MobileCarrierASNTable
	dcASN            *DCASNTable
	asnLookup        ASNLookup
}

func NewTCPMSSFilter(minMSS uint8) *TCPMSSFilter {
	if minMSS == 0 {
		minMSS = 2
	}
	return &TCPMSSFilter{minMSS: minMSS}
}

func (f *TCPMSSFilter) ConfigureTunnel(
	enabled bool,
	threshold uint16,
	mobileCarrierASN *MobileCarrierASNTable,
	dcASN *DCASNTable,
	lookup ASNLookup,
) {
	if f == nil {
		return
	}
	if threshold == 0 {
		threshold = 1400
	}
	f.tunnelEnabled = enabled
	f.tunnelThreshold = threshold
	f.mobileCarrierASN = mobileCarrierASN
	f.dcASN = dcASN
	f.asnLookup = lookup
}

func (f *TCPMSSFilter) Check(ctx context.Context, evt *domain.Event) error {
	if f == nil || evt == nil || evt.TCPMSSSet == 0 {
		return nil
	}
	if tcpMSSHighByte(evt.TCPMSS) < f.minMSS {
		metrics.TCPMSSAnomalyTotal.WithLabelValues("low_mss").Inc()
		addFraudSignal(evt, FraudReasonTCPMSSAnomaly)
	}
	f.checkTunnel(evt)
	return nil
}

func (f *TCPMSSFilter) checkTunnel(evt *domain.Event) {
	if !f.tunnelEnabled || f.asnLookup == nil || evt.IP == "" {
		return
	}
	if tcpMSSWireValue(evt.TCPMSS) >= f.tunnelThreshold {
		return
	}
	asn, ok := f.asnLookup.LookupASN(evt.IP)
	if !ok || asn == 0 {
		return
	}
	if f.mobileCarrierASN != nil && f.mobileCarrierASN.IsMobileCarrier(asn) {
		return
	}
	if f.dcASN != nil && f.dcASN.Ready() && f.dcASN.IsDatacenter(asn) {
		return
	}
	metrics.TCPMSSAnomalyTotal.WithLabelValues("tunnel_mss").Inc()
	addFraudSignal(evt, FraudReasonTCPTunnelMSS)
}

func tcpMSSHighByte(mss uint16) uint8 {
	if mss <= 255 {
		return uint8(mss)
	}
	return uint8(mss >> 8)
}

func tcpMSSWireValue(mss uint16) uint16 {
	if mss <= 255 {
		return mss << 8
	}
	return mss
}
