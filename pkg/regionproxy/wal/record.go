// Package wal implements the region-proxy mmap WAL used for upstream batch durability.
package wal

// HeaderSize is the cache-line padded WAL record header size (MULTI_REGION §0 / M2.1).
const HeaderSize = 128

const flagsOffset = 104

// FactorUOffset is the mmap byte offset of factor_u in a WAL record header.
const FactorUOffset = 16

// MaxPayloadSize is the largest ingress payload KeyGen accepts.
const MaxPayloadSize = 4096

// WAL lifecycle flags stored in the record header (bit masks).
const (
	WalFlagAppended       uint8 = 1 << 0
	WalFlagDedupReady     uint8 = 1 << 1
	WalFlagForwardClaimed uint8 = 1 << 2
	WalFlagRemoteAcked    uint8 = 1 << 3
)

// Header is a cache-line padded WAL record header.
type Header struct {
	Seq        uint64
	PayloadLen uint32
	_          uint32
	FactorU    [32]byte
	_          [56]byte
	Flags      uint8
	_          [23]byte
}

// Has reports whether flag is set in the header bitmask.
func (h *Header) Has(flag uint8) bool {
	return h.Flags&flag != 0
}

// Set sets one flag bit in the header bitmask.
func (h *Header) Set(flag uint8) {
	h.Flags |= flag
}
