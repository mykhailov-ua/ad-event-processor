package gnet

import "sync"

const maxPoolObjectSize = 64 * 1024

var requestBufferPool = sync.Pool{
	New: func() any {
		b := make([]byte, 4096)
		return &b
	},
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
