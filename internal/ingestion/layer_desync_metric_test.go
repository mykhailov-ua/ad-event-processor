package ingestion

import (
	"testing"

	"ad-event-processor/internal/ingestion/pb"
	"ad-event-processor/internal/metrics"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLayerDesyncMetric_holdoutFraudStreamConsumer(t *testing.T) {
	campaignID := uuid.New()
	pbEvt := &pb.AdStreamEvent{
		CampaignId:       campaignID[:],
		EventType:        []byte("click"),
		FraudReason:      []byte(FraudReasonCodeTCPSynOSMismatch),
		FraudScore:       70,
		LayerDesyncCount: 2,
	}
	raw, err := pbEvt.MarshalVT()
	require.NoError(t, err)

	before := testutil.ToFloat64(metrics.FraudStreamLayerDesyncTotal.WithLabelValues("2"))

	consumer := &StreamConsumer{}
	evt := consumer.parseMessage("1-0", map[string]interface{}{
		"d": raw,
	})
	require.NotNil(t, evt)
	assert.Equal(t, uint8(2), evt.LayerDesyncCount)
	assert.Equal(t, before+1, testutil.ToFloat64(metrics.FraudStreamLayerDesyncTotal.WithLabelValues("2")))
}

func TestLayerDesyncMetric_holdoutNonFraudSkipped(t *testing.T) {
	campaignID := uuid.New()
	pbEvt := &pb.AdStreamEvent{
		CampaignId: campaignID[:],
		EventType:  []byte("impression"),
	}
	raw, err := pbEvt.MarshalVT()
	require.NoError(t, err)

	before := testutil.ToFloat64(metrics.FraudStreamLayerDesyncTotal.WithLabelValues("0"))

	consumer := &StreamConsumer{}
	evt := consumer.parseMessage("1-0", map[string]interface{}{
		"d": raw,
	})
	require.NotNil(t, evt)
	assert.Equal(t, uint8(0), evt.LayerDesyncCount)
	assert.Equal(t, before, testutil.ToFloat64(metrics.FraudStreamLayerDesyncTotal.WithLabelValues("0")))
}

func TestLayerDesyncMetric_holdoutCapAtFive(t *testing.T) {
	campaignID := uuid.New()
	pbEvt := &pb.AdStreamEvent{
		CampaignId:       campaignID[:],
		EventType:        []byte("click"),
		FraudReason:      []byte("tcp_syn_os_mismatch,tls_ja4_mismatch"),
		FraudScore:       90,
		LayerDesyncCount: 9,
	}
	raw, err := pbEvt.MarshalVT()
	require.NoError(t, err)

	before := testutil.ToFloat64(metrics.FraudStreamLayerDesyncTotal.WithLabelValues("5"))

	consumer := &StreamConsumer{}
	evt := consumer.parseMessage("1-0", map[string]interface{}{
		"d": raw,
	})
	require.NotNil(t, evt)
	assert.Equal(t, uint8(9), evt.LayerDesyncCount)
	assert.Equal(t, before+1, testutil.ToFloat64(metrics.FraudStreamLayerDesyncTotal.WithLabelValues("5")))
}
