package ingest

import (
	"context"
	"testing"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/metrics"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseHTTP1_XTCPMSS(t *testing.T) {
	wire := []byte("POST /track HTTP/1.1\r\nX-TCP-MSS: 1\r\nContent-Length: 0\r\n\r\n")
	n, req, err := parseHTTP1(wire, 1024, nil)
	require.NoError(t, err)
	require.Equal(t, uint8(1), req.TCPMSSSet)
	assert.Equal(t, uint16(1), req.TCPMSS)
	assert.Equal(t, len(wire), n)
}

func TestParseHTTP1_XTCPMSSFullValue(t *testing.T) {
	wire := []byte("POST /track HTTP/1.1\r\nX-TCP-MSS: 1460\r\nContent-Length: 0\r\n\r\n")
	n, req, err := parseHTTP1(wire, 1024, nil)
	require.NoError(t, err)
	require.Equal(t, uint8(1), req.TCPMSSSet)
	assert.Equal(t, uint16(1460), req.TCPMSS)
	assert.Equal(t, len(wire), n)
}

func TestTCPMSSFilter_lowMSSSignal(t *testing.T) {
	before := testutil.ToFloat64(metrics.TCPMSSAnomalyTotal.WithLabelValues("low_mss"))
	f := NewTCPMSSFilter(2)
	evt := domain.EventPool.Get().(*domain.Event)
	defer domain.EventPool.Put(evt)
	evt.Reset()
	acc := attachFraudAccumulator(evt)
	defer releaseFraudAccumulator(evt, acc)
	evt.CampaignID = uuid.New()
	evt.TCPMSSSet = 1
	evt.TCPMSS = 1
	require.NoError(t, f.Check(context.Background(), evt))
	assert.True(t, acc.has(FraudReasonTCPMSSAnomaly))
	assert.Equal(t, before+1, testutil.ToFloat64(metrics.TCPMSSAnomalyTotal.WithLabelValues("low_mss")))
}

func TestTCPMSSFilter_normalMSSNoSignal(t *testing.T) {
	f := NewTCPMSSFilter(2)
	evt := domain.EventPool.Get().(*domain.Event)
	defer domain.EventPool.Put(evt)
	evt.Reset()
	acc := attachFraudAccumulator(evt)
	defer releaseFraudAccumulator(evt, acc)
	evt.TCPMSSSet = 1
	evt.TCPMSS = 5
	require.NoError(t, f.Check(context.Background(), evt))
	assert.False(t, acc.has(FraudReasonTCPMSSAnomaly))
}

func TestTCPMSSFilter_missingHeaderSkipped(t *testing.T) {
	f := NewTCPMSSFilter(2)
	evt := &domain.Event{TCPMSSSet: 0, TCPMSS: 1}
	require.NoError(t, f.Check(context.Background(), evt))
	assert.Empty(t, evt.FraudReason)
}

func TestTCPMSSAnomalyMetric_registered(t *testing.T) {
	require.NotNil(t, metrics.TCPMSSAnomalyTotal)
	_ = metrics.TCPMSSAnomalyTotal.WithLabelValues("low_mss")
	_ = metrics.TCPMSSAnomalyTotal.WithLabelValues("tunnel_mss")
}

type stubASNLookup map[string]uint32

func (s stubASNLookup) LookupASN(ip string) (uint32, bool) {
	asn, ok := s[ip]
	return asn, ok
}

const testResidentialASN uint32 = 12345

func TestTCPMSSTunnel_holdoutHomeFiberNormalMSSPasses(t *testing.T) {
	f := NewTCPMSSFilter(2)
	f.ConfigureTunnel(true, 1400, NewMobileCarrierASNTable(nil), nil, stubASNLookup{"10.0.0.1": testResidentialASN})
	evt := domain.EventPool.Get().(*domain.Event)
	defer domain.EventPool.Put(evt)
	evt.Reset()
	acc := attachFraudAccumulator(evt)
	defer releaseFraudAccumulator(evt, acc)
	evt.IP = "10.0.0.1"
	evt.TCPMSSSet = 1
	evt.TCPMSS = 1460
	require.NoError(t, f.Check(context.Background(), evt))
	assert.False(t, acc.has(FraudReasonTCPTunnelMSS))
}

func TestTCPMSSTunnel_holdoutLowMSSResidentialFlags(t *testing.T) {
	before := testutil.ToFloat64(metrics.TCPMSSAnomalyTotal.WithLabelValues("tunnel_mss"))
	f := NewTCPMSSFilter(2)
	f.ConfigureTunnel(true, 1400, NewMobileCarrierASNTable(nil), nil, stubASNLookup{"10.0.0.1": testResidentialASN})
	evt := domain.EventPool.Get().(*domain.Event)
	defer domain.EventPool.Put(evt)
	evt.Reset()
	acc := attachFraudAccumulator(evt)
	defer releaseFraudAccumulator(evt, acc)
	evt.IP = "10.0.0.1"
	evt.TCPMSSSet = 1
	evt.TCPMSS = 1280
	require.NoError(t, f.Check(context.Background(), evt))
	assert.True(t, acc.has(FraudReasonTCPTunnelMSS))
	assert.Equal(t, before+1, testutil.ToFloat64(metrics.TCPMSSAnomalyTotal.WithLabelValues("tunnel_mss")))
}

func TestTCPMSSTunnel_holdoutMobileCarrierSkips(t *testing.T) {
	f := NewTCPMSSFilter(2)
	f.ConfigureTunnel(true, 1400, NewMobileCarrierASNTable(nil), nil, stubASNLookup{"10.0.0.2": 21928})
	evt := domain.EventPool.Get().(*domain.Event)
	defer domain.EventPool.Put(evt)
	evt.Reset()
	acc := attachFraudAccumulator(evt)
	defer releaseFraudAccumulator(evt, acc)
	evt.IP = "10.0.0.2"
	evt.TCPMSSSet = 1
	evt.TCPMSS = 1280
	require.NoError(t, f.Check(context.Background(), evt))
	assert.False(t, acc.has(FraudReasonTCPTunnelMSS))
}

func TestTCPMSSTunnel_holdoutDatacenterSkips(t *testing.T) {
	dc := NewDCASNTable()
	dc.Publish(BuildDCASNSnapshot(map[uint32]struct{}{99999: {}}, 1))
	f := NewTCPMSSFilter(2)
	f.ConfigureTunnel(true, 1400, NewMobileCarrierASNTable(nil), dc, stubASNLookup{"10.0.0.3": 99999})
	evt := domain.EventPool.Get().(*domain.Event)
	defer domain.EventPool.Put(evt)
	evt.Reset()
	acc := attachFraudAccumulator(evt)
	defer releaseFraudAccumulator(evt, acc)
	evt.IP = "10.0.0.3"
	evt.TCPMSSSet = 1
	evt.TCPMSS = 1280
	require.NoError(t, f.Check(context.Background(), evt))
	assert.False(t, acc.has(FraudReasonTCPTunnelMSS))
}

func TestTCPMSSTunnel_holdoutMissingHeaderFailOpen(t *testing.T) {
	f := NewTCPMSSFilter(2)
	f.ConfigureTunnel(true, 1400, NewMobileCarrierASNTable(nil), nil, stubASNLookup{"10.0.0.1": testResidentialASN})
	evt := domain.EventPool.Get().(*domain.Event)
	defer domain.EventPool.Put(evt)
	evt.Reset()
	acc := attachFraudAccumulator(evt)
	defer releaseFraudAccumulator(evt, acc)
	evt.IP = "10.0.0.1"
	evt.TCPMSSSet = 0
	require.NoError(t, f.Check(context.Background(), evt))
	assert.False(t, acc.has(FraudReasonTCPTunnelMSS))
}

func TestTCPMSSTunnel_holdoutUnknownASNFailOpen(t *testing.T) {
	f := NewTCPMSSFilter(2)
	f.ConfigureTunnel(true, 1400, NewMobileCarrierASNTable(nil), nil, stubASNLookup{})
	evt := domain.EventPool.Get().(*domain.Event)
	defer domain.EventPool.Put(evt)
	evt.Reset()
	acc := attachFraudAccumulator(evt)
	defer releaseFraudAccumulator(evt, acc)
	evt.IP = "10.0.0.9"
	evt.TCPMSSSet = 1
	evt.TCPMSS = 1280
	require.NoError(t, f.Check(context.Background(), evt))
	assert.False(t, acc.has(FraudReasonTCPTunnelMSS))
}

func TestTCPMSSWireValue(t *testing.T) {
	assert.Equal(t, uint16(1280), tcpMSSWireValue(5))
	assert.Equal(t, uint16(1460), tcpMSSWireValue(1460))
}
