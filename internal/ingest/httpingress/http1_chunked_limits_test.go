package httpingress

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHTTP1ChunkedRejectMicroChunk(t *testing.T) {
	payload := []byte("POST /openrtb/bid HTTP/1.1\r\nTransfer-Encoding: chunked\r\n\r\n" +
		"1\r\n" +
		"a\r\n" +
		"0\r\n\r\n")
	_, _, err := ParseHTTP1Limits(payload, 1<<20, nil, ParseLimits{MinChunkedDataBytes: 64})
	require.ErrorIs(t, err, ErrInvalid)
}
