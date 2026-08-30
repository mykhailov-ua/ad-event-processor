package wal

const HeaderSize = 128 // mmap record header; multiple of 64 (cache line)

const flagsOffset = 104 // byte offset of Flags in on-disk header

const FactorUOffset = 16 // byte offset of 32-byte FactorU dedup key

const MaxPayloadSize = 4096 // max bytes after header per WAL record

const (
	WalFlagAppended       uint8 = 1 << 0 // record fully written past header+payload
	WalFlagDedupReady     uint8 = 1 << 1 // FactorU dedup slot populated
	WalFlagForwardClaimed uint8 = 1 << 2 // uplink forward lease claimed
	WalFlagRemoteAcked    uint8 = 1 << 3 // global control ack received
)

type Header struct {
	Seq        uint64   // [0:8] LE
	PayloadLen uint32   // [8:12] LE
	_          uint32   // [12:16] reserved
	FactorU    [32]byte // [16:48] dedup factor_u wire bytes
	_          [56]byte // [48:104] reserved
	Flags      uint8    // [104] lifecycle bits (WalFlag*)
	_          [23]byte // [105:128] reserved
}

func (h *Header) Has(flag uint8) bool {
	return h.Flags&flag != 0
}

func (h *Header) Set(flag uint8) {
	h.Flags |= flag
}
