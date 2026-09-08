package ingest

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func buildH2TrackRequest(body []byte) []byte {
	return buildH2TrackRequestOnStream(1, body, true)
}

func buildH2TrackRequestOnStream(streamID uint32, body []byte, withBootstrap bool) []byte {
	hdrBlock := []byte{0x83, 0x04, 0x06, '/', 't', 'r', 'a', 'c', 'k'}
	if len(body) > 0 {
		hdrBlock = append(hdrBlock, 0x9f)
	}

	var out []byte
	if withBootstrap {
		out = append(out, h2ClientPreface...)
		out = append(out, buildH2Frame(0, h2FrameSettings, 0, nil)...)
	}
	out = append(out, buildH2HeadersDataFrames(streamID, hdrBlock, body)...)
	return out
}

func buildH2PipelinedTrackRequests(n int, body []byte) []byte {
	var buf bytes.Buffer
	for i := range n {
		streamID := uint32(1 + i*2)
		buf.Write(buildH2TrackRequestOnStream(streamID, body, i == 0))
	}
	return buf.Bytes()
}

func TestHTTP2IngressParseTrack(t *testing.T) {
	body := []byte(`{"campaign_id":"00000000-0000-0000-0000-000000000001","type":"click"}`)
	wire := buildH2TrackRequest(body)
	st := newH2ConnState()
	consumed, req, streamID, settings, err := parseH2Ingress(wire, &st, 1<<20)
	require.NoError(t, err)
	assert.Greater(t, consumed, h2ClientPrefaceLen)
	assert.NotEmpty(t, settings)
	assert.Equal(t, uint32(1), streamID)
	assert.Equal(t, "POST", string(req.Method))
	assert.Equal(t, "/track", string(req.Path))
	assert.Equal(t, body, req.Body)
}

func TestHTTP2WrapResponse202(t *testing.T) {
	h1 := []byte("HTTP/1.1 202 Accepted\r\nContent-Type: application/json\r\nContent-Length: 2\r\n\r\n{}")
	dst := make([]byte, 512)
	n, err := h2WrapH1Response(dst, 1, h1)
	require.NoError(t, err)
	assert.Greater(t, n, h2FrameHeaderSize)
}
