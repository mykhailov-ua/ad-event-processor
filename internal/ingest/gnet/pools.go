package gnet

import "sync"

const maxPoolObjectSize = 64 * 1024

const MaxPoolObjectSize = maxPoolObjectSize

var RequestBufferPool = requestBufferPool

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
	if buf == nil || cap(*buf) > maxPoolObjectSize {
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
