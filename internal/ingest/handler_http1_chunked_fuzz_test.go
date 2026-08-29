package ingest

import (
	"testing"
)

func FuzzHTTP1Chunked(f *testing.F) {
	f.Add(fragmentedChunkedOpenRTBRequest())
	f.Add([]byte("POST /openrtb/bid HTTP/1.1\r\nTransfer-Encoding: chunked\r\n\r\n0\r\n\r\n"))

	f.Fuzz(func(t *testing.T, data []byte) {
		fuzzNoPanic(t, "parseChunkSizeLine", func() {
			_, _, _ = parseChunkSizeLine(data, 0, len(data))
		})
		var scratch []byte
		fuzzNoPanic(t, "parseHTTP1ChunkedBody", func() {
			_, _, _, _ = parseHTTP1ChunkedBody(data, 0, 1<<20, &scratch)
		})
	})
}
