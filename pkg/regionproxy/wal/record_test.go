package wal

import (
	"testing"
	"unsafe"
)

const cacheLineBytes = 64

func TestHeaderSizeCacheLine(t *testing.T) {
	if unsafe.Sizeof(Header{}) != HeaderSize {
		t.Fatalf("Header size = %d, want %d", unsafe.Sizeof(Header{}), HeaderSize)
	}
	if HeaderSize%cacheLineBytes != 0 {
		t.Fatalf("HeaderSize = %d, want multiple of %d", HeaderSize, cacheLineBytes)
	}
	if unsafe.Sizeof(Header{})%cacheLineBytes != 0 {
		t.Fatalf("Header struct size = %d, want multiple of %d", unsafe.Sizeof(Header{}), cacheLineBytes)
	}
}
