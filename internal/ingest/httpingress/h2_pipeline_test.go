package httpingress

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	SetIngressPathValidFn(func(method, path []byte) bool {
		if len(method) == 4 && method[0] == 'P' && method[1] == 'O' && method[2] == 'S' && method[3] == 'T' {
			return PathHasPrefix(path, "/track") || PathHasPrefix(path, "/openrtb/bid") || PathHasPrefix(path, "/tg/bid")
		}
		return false
	})
}

func h2Frame(streamID uint32, typ, flags byte, payload []byte) []byte {
	hdr := make([]byte, H2FrameHeaderSize)
	EncodeH2FrameHeader(hdr, uint32(len(payload)), typ, flags, streamID)
	hdr = append(hdr, payload...)
	return hdr
}

func h2HeadersDataFrames(streamID uint32, hdrBlock, body []byte) []byte {
	flags := byte(H2FlagEndHeaders)
	if len(body) == 0 {
		flags |= H2FlagEndStream
	}
	var out []byte
	out = append(out, h2Frame(streamID, H2FrameHeaders, flags, hdrBlock)...)
	if len(body) > 0 {
		out = append(out, h2Frame(streamID, H2FrameData, H2FlagEndStream, body)...)
	}
	return out
}

func h2TrackRequestOnStream(streamID uint32, body []byte, withBootstrap bool) []byte {
	hdrBlock := []byte{0x83, 0x04, 0x06, '/', 't', 'r', 'a', 'c', 'k'}
	if len(body) > 0 {
		hdrBlock = append(hdrBlock, 0x9f)
	}
	var out []byte
	if withBootstrap {
		out = append(out, H2ClientPreface...)
		out = append(out, h2Frame(0, H2FrameSettings, 0, nil)...)
	}
	out = append(out, h2HeadersDataFrames(streamID, hdrBlock, body)...)
	return out
}

func TestCountCompleteH2Messages_pipelineDepth(t *testing.T) {
	body := []byte(`{"campaign_id":"00000000-0000-0000-0000-000000000001","type":"click"}`)
	var wire []byte
	for i := range 3 {
		streamID := uint32(1 + i*2)
		wire = append(wire, h2TrackRequestOnStream(streamID, body, i == 0)...)
	}
	assert.Equal(t, 3, CountCompleteH2Messages(wire, 1<<20, nil))
}

func TestCountCompleteH2Messages_incompleteTail(t *testing.T) {
	body := []byte(`{"campaign_id":"00000000-0000-0000-0000-000000000001","type":"click"}`)
	wire := h2TrackRequestOnStream(1, body, true)
	tail := h2TrackRequestOnStream(3, body, false)
	wire = append(wire, tail[:H2FrameHeaderSize+2]...)
	assert.Equal(t, 1, CountCompleteH2Messages(wire, 1<<20, nil))
}

func TestCountCompleteH2Messages_establishedConn(t *testing.T) {
	body := []byte(`{"campaign_id":"00000000-0000-0000-0000-000000000001","type":"click"}`)
	first := h2TrackRequestOnStream(1, body, true)
	st := NewH2ConnState()
	_, _, _, _, err := ParseH2Ingress(first, &st, 1<<20)
	require.NoError(t, err)
	require.True(t, st.Established)

	second := h2TrackRequestOnStream(3, body, false)
	assert.Equal(t, 1, CountCompleteH2Messages(second, 1<<20, &st))
}
