package logger

import (
	"sync"
	"unsafe"
)

type AlignedBuffer struct {
	raw     []byte
	aligned []byte
	offset  int
}

func NewAlignedBuffer(size int) *AlignedBuffer {
	raw := make([]byte, size+4096)
	ptr := uintptr(unsafe.Pointer(&raw[0]))
	misalignment := ptr & 4095
	var offset uintptr
	if misalignment != 0 {
		offset = 4096 - misalignment
	}
	aligned := raw[offset : offset+uintptr(size)]
	return &AlignedBuffer{
		raw:     raw,
		aligned: aligned,
		offset:  0,
	}
}

func (b *AlignedBuffer) Write(data []byte) int {
	n := copy(b.aligned[b.offset:], data)
	b.offset += n
	return n
}

func (b *AlignedBuffer) WriteByte(c byte) error {
	b.aligned[b.offset] = c
	b.offset++
	return nil
}

func (b *AlignedBuffer) Reset() {
	b.offset = 0
}

func (b *AlignedBuffer) Bytes() []byte {
	return b.aligned[:b.offset]
}

func (b *AlignedBuffer) Available() int {
	return len(b.aligned) - b.offset
}

var bufferPool sync.Pool

func (l *Logger) getBuffer() *AlignedBuffer {
	val := bufferPool.Get()
	if val == nil {
		return NewAlignedBuffer(l.cfg.FlushBufferSize)
	}
	buf := val.(*AlignedBuffer)
	if len(buf.aligned) < l.cfg.FlushBufferSize {
		return NewAlignedBuffer(l.cfg.FlushBufferSize)
	}
	return buf
}
