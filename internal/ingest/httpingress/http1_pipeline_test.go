package httpingress

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCountCompleteHTTP1Messages_pipelineDepth(t *testing.T) {
	minimalPOST := []byte("POST /track HTTP/1.1\r\nContent-Length: 0\r\n\r\n")
	pipelined := bytes.Repeat(minimalPOST, 5)
	assert.Equal(t, 5, CountCompleteHTTP1Messages(pipelined, 1<<20, nil, ParseLimits{}))

	partial := append(pipelined, []byte("POST /track HTTP/1.1\r\n")...)
	assert.Equal(t, 5, CountCompleteHTTP1Messages(partial, 1<<20, nil, ParseLimits{}))
}

func TestCountCompleteHTTP1Messages_incompleteOnly(t *testing.T) {
	wire := []byte("POST /track HTTP/1.1\r\nContent-Length: 99\r\n\r\n")
	assert.Equal(t, 0, CountCompleteHTTP1Messages(wire, 1<<20, nil, ParseLimits{}))
}

func TestParseHTTP1_incompleteMidHeaderLine_holdout(t *testing.T) {
	full := []byte("POST /track HTTP/1.1\r\nContent-Type: application/json\r\nConnection: keep-alive\r\nContent-Length: 5\r\n\r\nhello")
	split := len(full) / 2
	_, _, err := ParseHTTP1(full[:split], 1<<20, nil)
	require.ErrorIs(t, err, ErrIncomplete)

	_, req, err := ParseHTTP1(full, 1<<20, nil)
	require.NoError(t, err)
	assert.Equal(t, "hello", string(req.Body))
}

func TestParseHTTP1_incompleteBodyAfterHeaders_holdout(t *testing.T) {
	hdr := []byte("POST /track HTTP/1.1\r\nContent-Type: application/json\r\nContent-Length: 5\r\n\r\n")
	wire := append(hdr, []byte("hel")...)
	var req Request
	consumed, err := ParseHTTP1LimitsInto(wire, 1<<20, nil, ParseLimits{}, &req)
	require.ErrorIs(t, err, ErrIncomplete)
	assert.Equal(t, len(hdr), consumed, "headers-complete partial body must report header offset as consumed")
}

func TestParseHTTP1Chunked_incompleteAfterHeaders_holdout(t *testing.T) {
	hdr := []byte("POST /openrtb/bid HTTP/1.1\r\nTransfer-Encoding: chunked\r\n\r\n")
	wire := append(hdr, []byte("5\r\nhel")...)
	consumed, _, _, err := ParseHTTP1ChunkedBody(wire, len(hdr), 1<<20, 0, nil)
	require.ErrorIs(t, err, ErrIncomplete)
	assert.Equal(t, len(hdr), consumed)
}

func TestHTTP1ChunkTrailers_capRejects(t *testing.T) {
	payload := []byte("POST /openrtb/bid HTTP/1.1\r\nTransfer-Encoding: chunked\r\n\r\n0\r\n")
	trailerLine := []byte("X-T: v\r\n")
	for range maxHTTP1ChunkTrailerLines + 1 {
		payload = append(payload, trailerLine...)
	}
	payload = append(payload, '\r', '\n')

	_, _, err := ParseHTTP1(payload, 1<<20, nil)
	require.ErrorIs(t, err, ErrInvalid)
}
