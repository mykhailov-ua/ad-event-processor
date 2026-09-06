package bufpool

import "sync"

// DefaultMaxCap matches ingest gnet requestBufferPool trim (64 KiB).
const DefaultMaxCap = 64 * 1024

var bytesPool = sync.Pool{
	New: func() any {
		b := make([]byte, 0, 512)
		return &b
	},
}

// GetBytes returns a pooled *[]byte with at least capHint capacity.
func GetBytes(capHint int) *[]byte {
	ptr := bytesPool.Get().(*[]byte)
	buf := *ptr
	if capHint > 0 && cap(buf) < capHint {
		*ptr = make([]byte, 0, capHint)
	} else {
		*ptr = buf[:0]
	}
	return ptr
}

// PutBytes returns buf to the pool when cap(buf) <= maxCap.
func PutBytes(buf *[]byte, maxCap int) {
	if buf == nil {
		return
	}
	if maxCap <= 0 {
		maxCap = DefaultMaxCap
	}
	if cap(*buf) > maxCap {
		return
	}
	*buf = (*buf)[:0]
	bytesPool.Put(buf)
}
