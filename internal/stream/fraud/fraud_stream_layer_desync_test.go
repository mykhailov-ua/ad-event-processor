package fraud

import (
	"testing"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/ingest/pb"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarshalFraudEventPayload_layerDesyncCount(t *testing.T) {
	evt := &domain.Event{
		ClickID:          "click-1",
		CampaignID:       uuid.New(),
		Type:             "click",
		FraudReason:      "tcp_syn_os_mismatch,tls_ja4_mismatch",
		FraudScore:       70,
		LayerDesyncCount: 2,
	}
	data, cleanup := MarshalFraudEventPayloadForTest(evt)
	require.NotNil(t, data)
	defer cleanup()

	pbEvt := &pb.AdStreamEvent{}
	require.NoError(t, pbEvt.UnmarshalVT(data))
	assert.Equal(t, uint32(2), pbEvt.LayerDesyncCount)
}
