package httpingress

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTP1TrackEdgePolicy_negativeMatrix(t *testing.T) {
	t.Run("missing_content_length", func(t *testing.T) {
		req := Request{Method: []byte("POST"), Path: []byte("/track")}
		err := http1TrackEdgePolicy(&req, 0)
		require.ErrorIs(t, err, ErrInvalid)
	})

	t.Run("chunked_transfer_encoding", func(t *testing.T) {
		req := Request{Method: []byte("POST"), Path: []byte("/track")}
		err := http1TrackEdgePolicy(&req, http1flHasTE|http1flChunkedTE|http1flCLSet)
		require.ErrorIs(t, err, ErrInvalid)
	})

	t.Run("openrtb_chunked_allowed_before_policy", func(t *testing.T) {
		req := Request{Method: []byte("POST"), Path: []byte("/openrtb/bid")}
		err := http1TrackEdgePolicy(&req, http1flChunkedTE)
		require.NoError(t, err)
	})

	t.Run("track_with_cl_ok", func(t *testing.T) {
		req := Request{Method: []byte("POST"), Path: []byte("/track")}
		err := http1TrackEdgePolicy(&req, http1flCLSet)
		require.NoError(t, err)
	})

	t.Run("track_rejects_gzip_transfer_encoding", func(t *testing.T) {
		req := Request{Method: []byte("POST"), Path: []byte("/track")}
		err := http1TrackEdgePolicy(&req, http1flHasTE|http1flInvalidTE|http1flCLSet)
		require.ErrorIs(t, err, ErrInvalid)
	})
}

func TestParseHTTP1_trackNegativeWire_holdout(t *testing.T) {
	rejectCases := []struct {
		name string
		wire []byte
	}{
		{name: "no_cl", wire: []byte("POST /track HTTP/1.1\r\nHost: t\r\n\r\n")},
		{name: "chunked_te", wire: []byte("POST /track HTTP/1.1\r\nHost: t\r\nTransfer-Encoding: chunked\r\n\r\n0\r\n\r\n")},
		{name: "cl_and_chunked", wire: []byte("POST /track HTTP/1.1\r\nHost: t\r\nContent-Length: 0\r\nTransfer-Encoding: chunked\r\n\r\n0\r\n\r\n")},
		{name: "double_te", wire: []byte("POST /track HTTP/1.1\r\nHost: t\r\nTransfer-Encoding: chunked\r\nTransfer-Encoding: gzip\r\n\r\n0\r\n\r\n")},
		{name: "gzip_te_with_cl", wire: []byte("POST /track HTTP/1.1\r\nHost: t\r\nTransfer-Encoding: gzip\r\nContent-Length: 0\r\n\r\n")},
	}
	for _, tc := range rejectCases {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := ParseHTTP1(tc.wire, 1<<20, nil)
			require.ErrorIs(t, err, ErrInvalid)
		})
	}
}

func FuzzParseHTTP1TrackWire(f *testing.F) {
	f.Add([]byte("POST /track HTTP/1.1\r\nHost: t\r\nContent-Length: 5\r\n\r\nhello"))
	f.Add([]byte("POST /track HTTP/1.1\r\nHost: t\r\nTransfer-Encoding: chunked\r\n\r\n0\r\n\r\n"))
	f.Add([]byte("POST /track HTTP/1.1\r\nHost: t\r\n\r\n"))

	f.Fuzz(func(t *testing.T, wire []byte) {
		if len(wire) > 1<<16 {
			t.Skip()
		}
		_, _, _ = ParseHTTP1(wire, 1<<20, nil)
	})
}

func TestHTTP1AssignTransferEncoding_doubleTE_invalid(t *testing.T) {
	var flags uint8
	require.NoError(t, http1AssignTransferEncoding(&flags, []byte("chunked")))
	err := http1AssignTransferEncoding(&flags, []byte("gzip"))
	require.ErrorIs(t, err, ErrInvalid)
	assert.True(t, flags&http1flChunkedTE != 0)
}
