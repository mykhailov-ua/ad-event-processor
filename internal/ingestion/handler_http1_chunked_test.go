package ingestion

import (
	"fmt"
	"strconv"
	"testing"

	"ad-event-processor/pkg/faultproof"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTP1ChunkedBody(t *testing.T) {
	const maxBody = int64(1024 * 1024)
	body := []byte(`{"id":"req-1","imp":[{"id":"1"}]}`)
	chunked := append([]byte(
		"POST /openrtb/bid HTTP/1.1\r\n"+
			"Transfer-Encoding: chunked\r\n"+
			"Content-Type: application/json\r\n"+
			"\r\n"),
		[]byte(fmt.Sprintf("%x\r\n", len(body)))...)
	chunked = append(chunked, body...)
	chunked = append(chunked, "\r\n0\r\n\r\n"...)
	n, req, err := parseHTTP1(chunked, maxBody, nil)
	require.NoError(t, err)
	assert.Equal(t, len(chunked), n)
	assert.Equal(t, body, req.Body)
	assert.Equal(t, len(body), req.ContentLength)
}

func TestHTTP1ChunkedTrackRejected(t *testing.T) {
	payload := []byte("POST /track HTTP/1.1\r\nTransfer-Encoding: chunked\r\n\r\n0\r\n\r\n")
	_, _, err := parseHTTP1(payload, 1024, nil)
	assert.ErrorIs(t, err, errInvalidRequest)
}

func TestHTTP1ChunkedEmpty(t *testing.T) {
	payload := []byte("POST /openrtb/bid HTTP/1.1\r\nTransfer-Encoding: chunked\r\n\r\n0\r\n\r\n")
	n, req, err := parseHTTP1(payload, 1024, nil)
	require.NoError(t, err)
	assert.Equal(t, len(payload), n)
	assert.Empty(t, req.Body)
}

func TestHTTP1ChunkedRejectCLCombo(t *testing.T) {
	payload := []byte("POST /track HTTP/1.1\r\nTransfer-Encoding: chunked\r\nContent-Length: 0\r\n\r\n0\r\n\r\n")
	_, _, err := parseHTTP1(payload, 1024, nil)
	assert.ErrorIs(t, err, errInvalidRequest)
}

func TestHTTP1ChunkedRejectExtension(t *testing.T) {
	payload := []byte("POST /track HTTP/1.1\r\nTransfer-Encoding: chunked\r\n\r\n5;ext=bad\r\n00000\r\nhello\r\n0\r\n\r\n")
	_, _, err := parseHTTP1(payload, 1<<20, nil)
	assert.ErrorIs(t, err, errInvalidRequest)
}

func TestChaos_ChunkExt_CRLFInExtension(t *testing.T) {
	wire := []byte("POST /track HTTP/1.1\r\nTransfer-Encoding: chunked\r\n\r\n" +
		"5;foo\r\n" +
		"5\r\n" +
		"hello\r\n" +
		"0\r\n\r\n")
	_, _, err := parseHTTP1(wire, 1<<20, nil)
	require.ErrorIs(t, err, errInvalidRequest)
}

func TestHTTP1ChunkedRejectOversizedHexSize(t *testing.T) {
	payload := []byte("POST /openrtb/bid HTTP/1.1\r\nTransfer-Encoding: chunked\r\n\r\n" +
		"ffffffffffffffff\r\n" +
		"0\r\n\r\n")
	_, _, err := parseHTTP1(payload, 1<<20, nil)
	assert.ErrorIs(t, err, errInvalidRequest)
}

func TestHTTP1ChunkedScratch_ShrinkOnClose(t *testing.T) {
	var scratch []byte
	_ = growChunkScratch(&scratch, 150*1024)
	require.GreaterOrEqual(t, cap(scratch), 150*1024)

	resetChunkScratch(&scratch)
	require.LessOrEqual(t, cap(scratch), chunkScratchRetainCap)

	faultproof.Log(t, "parser_security_ps_h03", map[string]string{
		"gap_id":    "chunked_extension_reject",
		"gap":       "closed",
		"cap_after": strconv.Itoa(cap(scratch)),
	})
}

func TestHTTP1ChunkedFragmentedZeroAlloc(t *testing.T) {
	const maxBody = int64(1024 * 1024)
	wire := fragmentedChunkedOpenRTBRequest()
	var scratch []byte
	allocs := testing.AllocsPerRun(100, func() {
		_, _, err := parseHTTP1(wire, maxBody, &scratch)
		require.NoError(t, err)
	})
	assert.Equal(t, float64(0), allocs)
}

func BenchmarkHTTP1ChunkedFragmented(b *testing.B) {
	const maxBody = int64(1024 * 1024)
	wire := fragmentedChunkedOpenRTBRequest()
	var scratch []byte
	b.ReportAllocs()
	for b.Loop() {
		_, _, err := parseHTTP1(wire, maxBody, &scratch)
		if err != nil {
			b.Fatal(err)
		}
	}
}
