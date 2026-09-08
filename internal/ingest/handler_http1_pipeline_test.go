package ingest

import (
	"bytes"
	"testing"

	"ad-event-processor/internal/config"

	"github.com/panjf2000/gnet/v2"
	"github.com/stretchr/testify/require"
)

func TestHTTP1Pipeline_DepthCloses(t *testing.T) {
	cfg := &config.Config{
		MaxRequestBodySize:    1 << 20,
		HTTP1MaxPipelineDepth: 2,
	}
	h := NewAdsPacketHandler(cfg, &mockRegistry{}, nil, nil, nil, NewJumpHashSharder(1), "fraud", nil)

	minimal := []byte("POST /track HTTP/1.1\r\nContent-Length: 0\r\n\r\n")
	conn := NewGnetHarnessConn(bytes.Repeat(minimal, 3))
	act := h.OnTraffic(conn)
	require.Equal(t, gnet.Close, act)
}

func TestHTTP1Pipeline_BusyDepthCloses(t *testing.T) {
	cfg := &config.Config{
		MaxRequestBodySize:    1 << 20,
		HTTP1MaxPipelineDepth: 2,
	}
	h := NewAdsPacketHandler(cfg, &mockRegistry{}, nil, nil, nil, NewJumpHashSharder(1), "fraud", nil)

	minimal := []byte("POST /track HTTP/1.1\r\nContent-Length: 0\r\n\r\n")
	conn := NewGnetHarnessConn(bytes.Repeat(minimal, 3))
	ctx := h.AllocConnContext(conn)
	ctx.HTTP1OffloadBusy.Store(true)
	conn.SetContext(ctx)

	act := h.OnTraffic(conn)
	require.Equal(t, gnet.Close, act)
}

func TestHTTP1Pipeline_BusyBufferCloses(t *testing.T) {
	cfg := &config.Config{
		MaxRequestBodySize:        1 << 20,
		HTTP1MaxPipelineBusyBytes: 128,
	}
	h := NewAdsPacketHandler(cfg, &mockRegistry{}, nil, nil, nil, NewJumpHashSharder(1), "fraud", nil)

	flood := bytes.Repeat([]byte("POST /track HTTP/1.1\r\nContent-Length: 0\r\n\r\n"), 10)
	conn := NewGnetHarnessConn(flood)
	ctx := h.AllocConnContext(conn)
	ctx.HTTP1OffloadBusy.Store(true)
	conn.SetContext(ctx)

	act := h.OnTraffic(conn)
	require.Equal(t, gnet.Close, act)
}
