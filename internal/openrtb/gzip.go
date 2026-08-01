package openrtb

import (
	"compress/gzip"
	"io"
	"sync"
)

// HTTPWriteOpts controls optional gzip on OpenRTB HTTP responses (T5).
type HTTPWriteOpts struct {
	Gzip bool
}

type sliceGzipWriter struct {
	dst []byte
	n   int
}

func (w *sliceGzipWriter) Write(p []byte) (int, error) {
	if w.n+len(p) > len(w.dst) {
		return 0, ErrBodyTooLarge
	}
	copy(w.dst[w.n:], p)
	w.n += len(p)
	return len(p), nil
}

var gzipWriterPool sync.Pool

func init() {
	gzipWriterPool.New = func() any {
		w, err := gzip.NewWriterLevel(io.Discard, gzip.BestSpeed)
		if err != nil {
			return gzip.NewWriter(io.Discard)
		}
		return w
	}
	var dst [256]byte
	var src [128]byte
	_, _ = gzipCompressInto(dst[:], src[:])
}

func gzipCompressInto(dst, src []byte) (int, error) {
	if len(src) == 0 {
		return 0, nil
	}
	sw := &sliceGzipWriter{dst: dst}
	zw := gzipWriterPool.Get().(*gzip.Writer)
	zw.Reset(sw)
	if _, err := zw.Write(src); err != nil {
		gzipWriterPool.Put(zw)
		return 0, err
	}
	if err := zw.Close(); err != nil {
		gzipWriterPool.Put(zw)
		return 0, err
	}
	gzipWriterPool.Put(zw)
	return sw.n, nil
}

// gzipMinBody is the minimum JSON body size to prefer gzip on the exchange path.
const gzipMinBody = 64

func shouldGzipBody(bodyLen int, opts HTTPWriteOpts) bool {
	return opts.Gzip && bodyLen >= gzipMinBody
}

var _ io.Writer = (*sliceGzipWriter)(nil)
