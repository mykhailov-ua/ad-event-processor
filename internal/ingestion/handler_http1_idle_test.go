package ingestion

import (
	"testing"
	"time"

	"espx/internal/config"

	"github.com/panjf2000/gnet/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTP1Incomplete_SpinClosesAfterMax(t *testing.T) {
	cfg := &config.Config{
		MaxRequestBodySize: 1 << 20,
		HTTP1IncompleteMax: 3,
		HTTP1BodyIdleMs:    60_000,
	}
	h := NewAdsPacketHandler(cfg, &mockRegistry{}, nil, nil, nil, NewJumpHashSharder(1), "fraud", nil)

	hdr := []byte("POST /track HTTP/1.1\r\nContent-Type: application/json\r\nContent-Length: 1048576\r\n\r\n")
	body := []byte(`{"type":"click","campaign_id":"550e8400-e29b-41d4-a716-446655440000","payload":{`)

	conn := newFaultGnetConn()
	conn.Append(hdr)
	require.Equal(t, gnet.None, h.OnTraffic(conn))

	var act gnet.Action
	for i := 0; i < len(body)+2; i++ {
		end := i + 1
		if end > len(body) {
			end = len(body)
		}
		if i < len(body) {
			conn.Append(body[i:end])
		}
		act = h.OnTraffic(conn)
		if act == gnet.Close {
			break
		}
	}
	require.Equal(t, gnet.Close, act, "slow drip must close after HTTP1_INCOMPLETE_MAX incomplete passes")
}

func TestHTTP1Incomplete_IdleClosesAfterDeadline(t *testing.T) {
	cfg := &config.Config{
		MaxRequestBodySize: 1 << 20,
		HTTP1IncompleteMax: 100,
		HTTP1BodyIdleMs:    1,
	}
	h := NewAdsPacketHandler(cfg, &mockRegistry{}, nil, nil, nil, NewJumpHashSharder(1), "fraud", nil)

	hdr := []byte("POST /track HTTP/1.1\r\nContent-Type: application/json\r\nContent-Length: 999\r\n\r\n")
	conn := newFaultGnetConn()
	conn.Append(hdr)
	require.Equal(t, gnet.None, h.OnTraffic(conn))

	time.Sleep(3 * time.Millisecond)
	act := h.OnTraffic(conn)
	require.Equal(t, gnet.Close, act)
}

func TestHTTP1Incomplete_BufferCapCloses(t *testing.T) {
	cfg := &config.Config{
		MaxRequestBodySize: 64,
		HTTP1IncompleteMax: 100,
		HTTP1BodyIdleMs:    60_000,
	}
	h := NewAdsPacketHandler(cfg, &mockRegistry{}, nil, nil, nil, NewJumpHashSharder(1), "fraud", nil)

	// Headers + partial body exceeding max buffered (64 + 8192 overhead still allows headers; use huge incomplete body buffer)
	payload := make([]byte, 0, 9000)
	payload = append(payload, []byte("POST /track HTTP/1.1\r\nContent-Length: 99999\r\n\r\n")...)
	payload = append(payload, make([]byte, 9000-len(payload))...)

	conn := newFaultGnetConn()
	conn.Append(payload)
	act := h.OnTraffic(conn)
	require.Equal(t, gnet.Close, act)
}

func TestHTTP1Incomplete_ResetsOnCompleteRequest(t *testing.T) {
	cfg := &config.Config{
		MaxRequestBodySize: 1 << 20,
		HTTP1IncompleteMax: 2,
		HTTP1BodyIdleMs:    60_000,
	}
	h := NewAdsPacketHandler(cfg, &mockRegistry{}, nil, nil, nil, NewJumpHashSharder(1), "fraud", nil)

	body := []byte(`{"campaign_id":"550e8400-e29b-41d4-a716-446655440000","type":"click"}`)
	req := BuildGnetPostTrackJSON(body)
	split := len(req) / 2

	conn := NewGnetHarnessConn(req[:split])
	assert.Equal(t, gnet.None, h.OnTraffic(conn))

	conn.inbound = append(conn.inbound[:0], req...)
	assert.Equal(t, gnet.None, h.OnTraffic(conn))
	assert.Equal(t, 1, conn.WriteCount())
}

func TestHTTP1HeadersComplete(t *testing.T) {
	assert.False(t, http1HeadersComplete([]byte("POST /track HTTP/1.1\r\nContent-Length: 1\r\n")))
	assert.True(t, http1HeadersComplete([]byte("POST /track HTTP/1.1\r\nContent-Length: 1\r\n\r\n")))
}
