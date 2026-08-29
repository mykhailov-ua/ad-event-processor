package stream

import (
	"sync/atomic"

	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
)

const (
	localQuantaCacheLine = 64
	localQuantaSlotCount = 4096
	localQuantaSlotMask  = localQuantaSlotCount - 1
)

const (
	LocalQuantaOff    uint32 = 0
	LocalQuantaShadow uint32 = 1
	LocalQuantaLive   uint32 = 2
)

type LocalQuantaCell struct {
	campaignHash uint32
	_            uint32
	remaining    atomic.Int64
	chunkSize    int64
	rpsEMA       atomic.Uint64
	campaignID   uuid.UUID
	_            [localQuantaCacheLine - 8 - 8 - 8 - 8 - 16]byte
}

type LocalQuantaLedger struct {
	cells [localQuantaSlotCount]LocalQuantaCell
	mode  atomic.Uint32
}

func NewLocalQuantaLedger() *LocalQuantaLedger {
	return &LocalQuantaLedger{}
}

func (l *LocalQuantaLedger) SetMode(mode string) {
	switch mode {
	case "shadow":
		l.mode.Store(LocalQuantaShadow)
	case "live":
		l.mode.Store(LocalQuantaLive)
	default:
		l.mode.Store(LocalQuantaOff)
	}
}

func (l *LocalQuantaLedger) Mode() uint32 {
	return l.mode.Load()
}

func ledgerCellHash(id uuid.UUID, subSlot int) uint32 {
	h := domain.CRC32Castagnoli(&id)
	if subSlot > 0 {
		h ^= uint32(subSlot) * 0x85ebca6b
	}
	return h
}

func (l *LocalQuantaLedger) cellFor(id uuid.UUID) (*LocalQuantaCell, uint32) {
	return l.cellForDebit(id, 0)
}

func (l *LocalQuantaLedger) cellForDebit(id uuid.UUID, subSlot int) (*LocalQuantaCell, uint32) {
	base := domain.CRC32Castagnoli(&id)
	sub := subSlot & 3
	idx := (base + uint32(sub)*1024) & localQuantaSlotMask
	return &l.cells[idx], ledgerCellHash(id, sub)
}

func (l *LocalQuantaLedger) HasCredit(id uuid.UUID) bool {
	cell, h := l.cellFor(id)
	return cell.campaignHash == h && cell.remaining.Load() > 0
}

func (l *LocalQuantaLedger) Remaining(id uuid.UUID) int64 {
	cell, h := l.cellFor(id)
	if cell.campaignHash != h {
		return 0
	}
	return cell.remaining.Load()
}

func (l *LocalQuantaLedger) ChunkSize(id uuid.UUID) int64 {
	cell, h := l.cellFor(id)
	if cell.campaignHash != h {
		return 0
	}
	return cell.chunkSize
}

func (l *LocalQuantaLedger) Refund(id uuid.UUID, amountMicro int64) {
	l.RefundDebit(id, 0, amountMicro)
}

func (l *LocalQuantaLedger) RefundDebit(id uuid.UUID, subSlot int, amountMicro int64) {
	if amountMicro <= 0 {
		return
	}
	cell, h := l.cellForDebit(id, subSlot)
	if cell.campaignHash != h {
		return
	}
	cell.remaining.Add(amountMicro)
}

func (l *LocalQuantaLedger) TrySpendLocal(id uuid.UUID, amountMicro int64) bool {
	return l.TrySpendDebit(id, 0, amountMicro)
}

func (l *LocalQuantaLedger) TrySpendDebit(id uuid.UUID, subSlot int, amountMicro int64) bool {
	if amountMicro <= 0 {
		return true
	}
	cell, h := l.cellForDebit(id, subSlot)
	if cell.campaignHash != h {
		return false
	}
	for {
		rem := cell.remaining.Load()
		if rem < amountMicro {
			return false
		}
		if cell.remaining.CompareAndSwap(rem, rem-amountMicro) {
			l.recordSpendEMA(cell)
			return true
		}
	}
}

func (l *LocalQuantaLedger) Credit(id uuid.UUID, amountMicro, chunkSize int64) {
	l.CreditDebit(id, 0, amountMicro, chunkSize)
}

func (l *LocalQuantaLedger) CreditDebit(id uuid.UUID, subSlot int, amountMicro, chunkSize int64) {
	if amountMicro <= 0 {
		return
	}
	cell, h := l.cellForDebit(id, subSlot)
	cell.campaignHash = h
	cell.campaignID = id
	if chunkSize > 0 {
		cell.chunkSize = chunkSize
	}
	cell.remaining.Add(amountMicro)
}

func (l *LocalQuantaLedger) NeedsRefill(id uuid.UUID, thresholdPct int) bool {
	cell, h := l.cellFor(id)
	if cell.campaignHash != h || cell.chunkSize <= 0 {
		return true
	}
	rem := cell.remaining.Load()
	if rem <= 0 {
		return true
	}
	if thresholdPct <= 0 {
		thresholdPct = 20
	}
	threshold := cell.chunkSize * int64(thresholdPct) / 100
	return rem < threshold
}

func (l *LocalQuantaLedger) RPSEMA(id uuid.UUID) float64 {
	cell, h := l.cellFor(id)
	if cell.campaignHash != h {
		return 0
	}
	return float64(cell.rpsEMA.Load()) / 1000.0
}

func (l *LocalQuantaLedger) recordSpendEMA(cell *LocalQuantaCell) {
	const alphaMilli = 100
	prev := cell.rpsEMA.Load()
	next := prev + alphaMilli
	if next > 1_000_000 {
		next = 1_000_000
	}
	cell.rpsEMA.Store(next)
}

func AdaptiveChunkSize(emaRPS float64, floorMicro, ceilingMicro, baseChunk int64) int64 {
	if baseChunk <= 0 {
		baseChunk = 5_000_000
	}
	if floorMicro <= 0 {
		floorMicro = 500_000
	}
	if ceilingMicro <= 0 {
		ceilingMicro = 50_000_000
	}
	chunk := baseChunk
	if emaRPS > 0 {
		scaled := int64(emaRPS * 10_000)
		if scaled > 0 {
			chunk = scaled
		}
	}
	if chunk < floorMicro {
		chunk = floorMicro
	}
	if chunk > ceilingMicro {
		chunk = ceilingMicro
	}
	return chunk
}
