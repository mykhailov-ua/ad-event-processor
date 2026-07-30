package wal

const HeaderSize = 128

const flagsOffset = 104

const FactorUOffset = 16

const MaxPayloadSize = 4096

const (
	WalFlagAppended       uint8 = 1 << 0
	WalFlagDedupReady     uint8 = 1 << 1
	WalFlagForwardClaimed uint8 = 1 << 2
	WalFlagRemoteAcked    uint8 = 1 << 3
)

type Header struct {
	Seq        uint64
	PayloadLen uint32
	_          uint32
	FactorU    [32]byte
	_          [56]byte
	Flags      uint8
	_          [23]byte
}

func (h *Header) Has(flag uint8) bool {
	return h.Flags&flag != 0
}

func (h *Header) Set(flag uint8) {
	h.Flags |= flag
}
