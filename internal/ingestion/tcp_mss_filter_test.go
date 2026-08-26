package ingestion

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
	assert.Equal(t, uint8(1), req.TCPMSS)
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
}
