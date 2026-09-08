package ingest

import (
	"testing"

	"ad-event-processor/internal/config"

	"github.com/panjf2000/gnet/v2"
	"github.com/stretchr/testify/require"
)

func TestH2Pipeline_DepthCloses(t *testing.T) {
	cfg := &config.Config{
		MaxRequestBodySize:    1 << 20,
		HTTP1MaxPipelineDepth: 2,
	}
	h := NewAdsPacketHandler(cfg, &mockRegistry{}, nil, nil, nil, NewJumpHashSharder(1), "fraud", nil)

	body := []byte(`{"campaign_id":"00000000-0000-0000-0000-000000000001","type":"click"}`)
	wire := buildH2PipelinedTrackRequests(3, body)
	conn := NewGnetHarnessConn(wire)

	act := h.onTrafficH2(conn, wire)
	require.Equal(t, gnet.Close, act)
}

func TestH2Pipeline_BufferCloses(t *testing.T) {
	cfg := &config.Config{
		MaxRequestBodySize:        1 << 20,
		HTTP1MaxPipelineBusyBytes: 256,
	}
	h := NewAdsPacketHandler(cfg, &mockRegistry{}, nil, nil, nil, NewJumpHashSharder(1), "fraud", nil)

	body := []byte(`{"campaign_id":"00000000-0000-0000-0000-000000000001","type":"click"}`)
	wire := buildH2PipelinedTrackRequests(5, body)
	conn := NewGnetHarnessConn(wire)

	act := h.onTrafficH2(conn, wire)
	require.Equal(t, gnet.Close, act)
}

func TestH2Pipeline_WithinDepthAccepts(t *testing.T) {
	cfg := &config.Config{
		MaxRequestBodySize:    1 << 20,
		HTTP1MaxPipelineDepth: 4,
	}
	h := NewAdsPacketHandler(cfg, &mockRegistry{}, nil, nil, nil, NewJumpHashSharder(1), "fraud", nil)

	body := []byte(`{"campaign_id":"00000000-0000-0000-0000-000000000001","type":"click"}`)
	wire := buildH2PipelinedTrackRequests(2, body)
	conn := NewGnetHarnessConn(wire)

	require.Equal(t, gnet.None, h.onTrafficH2(conn, wire))
}
