package gnet

import (
	"sync"

	"ad-event-processor/internal/metrics"
)

const maxPoolObjectSize = 64 * 1024

const MaxPoolObjectSize = maxPoolObjectSize

func GetRequestBuffer() *[]byte {
	return requestBufferPool.Get().(*[]byte)
}

var requestBufferPool = sync.Pool{
	New: func() any {
		b := make([]byte, 4096)
		return &b
	},
}

func PutRequestBuffer(buf *[]byte) {
	putRequestBuffer(buf)
}

func putRequestBuffer(buf *[]byte) {
	if buf == nil {
		return
	}
	if cap(*buf) > maxPoolObjectSize {
		metrics.RequestBufferPoolTrimTotal.Inc()
		return
	}
	*buf = (*buf)[:0]
	requestBufferPool.Put(buf)
}

var responseBytesPool = sync.Pool{
	New: func() any {
		s := make([]byte, 4096)
		return &s
	},
}
