package ingest

import (
	"testing"

	"ad-event-processor/internal/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseHTTP1_connTimingHeaders(t *testing.T) {
	wire := []byte("POST /track HTTP/1.1\r\nX-RTT-SYN-MS: 42\r\nX-TTFB-APP-MS: 210\r\nContent-Length: 0\r\n\r\n")
	_, req, err := parseHTTP1(wire, 1<<20, nil)
	require.NoError(t, err)
	require.Equal(t, uint8(connTimingRTTBit|connTimingTTFBBit), req.ConnTimingSet)
	assert.Equal(t, uint16(42), req.RTTSynMS)
	assert.Equal(t, uint16(210), req.TTFBAppMS)
}

func TestRTTSplitDelta_holdoutSubtractsSynRTT(t *testing.T) {
	assert.Equal(t, uint16(168), rttSplitDeltaMS(42, 210))
	assert.Equal(t, uint16(0), rttSplitDeltaMS(210, 42))
}

func TestFillConnTimingFromRequest_holdoutSetsEventDelta(t *testing.T) {
	req := parsedHTTPRequest{
		RTTSynMS:      50,
		TTFBAppMS:     300,
		ConnTimingSet: connTimingRTTBit | connTimingTTFBBit,
	}
	evt := domain.EventPool.Get().(*domain.Event)
	defer domain.EventPool.Put(evt)
	evt.Reset()
	fillConnTimingFromRequest(evt, &req)
	assert.Equal(t, uint8(connTimingRTTBit|connTimingTTFBBit), evt.ConnTimingSet)
	assert.Equal(t, uint16(50), evt.RTTSynMS)
	assert.Equal(t, uint16(300), evt.TTFBAppMS)
	assert.Equal(t, uint16(250), evt.RTTSplitDeltaMS)
}
