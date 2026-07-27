package wal

import (
	"testing"
	"unsafe"
)

func TestHeaderSizeCacheLine(t *testing.T) {
	if unsafe.Sizeof(Header{}) != HeaderSize {
		t.Fatalf("Header size = %d, want %d", unsafe.Sizeof(Header{}), HeaderSize)
	}
}
