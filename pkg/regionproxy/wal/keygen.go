package wal

import (
	"context"
	"fmt"
	"time"

	"ad-event-processor/pkg/iogate"
)

func (w *WAL) KeyGenQueueDepth() int64 {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.keyGenQueueDepthLocked()
}

func (w *WAL) keyGenQueueDepthLocked() int64 {
	pos := int64(0)
	writePos := w.writePos.Load()
	var pending int64
	for pos < writePos {
		if len(w.mmap) <= int(pos)+HeaderSize {
			break
		}
		hdr := readHeader(w.mmap[pos:])
		if hdr.Seq == 0 && hdr.PayloadLen == 0 && hdr.Flags == 0 {
			break
		}
		if !hdr.Has(WalFlagAppended) {
			break
		}
		payloadLen := int64(hdr.PayloadLen)
		recordLen := int64(HeaderSize) + payloadLen
		if pos+recordLen > writePos {
			break
		}
		if !hdr.Has(WalFlagDedupReady) {
			pending++
		}
		pos += recordLen
	}
	return pending
}

func (w *WAL) ProcessPendingKeyGen(maxRecords int, derive func(seq uint64, payload []byte) ([32]byte, error)) (int, error) {
	if maxRecords <= 0 {
		return 0, nil
	}
	w.mu.Lock()
	defer w.mu.Unlock()

	processed := 0
	pos := int64(0)
	writePos := w.writePos.Load()
	for processed < maxRecords && pos < writePos {
		if len(w.mmap) <= int(pos)+HeaderSize {
			break
		}
		hdr := readHeader(w.mmap[pos:])
		if hdr.Seq == 0 && hdr.PayloadLen == 0 && hdr.Flags == 0 {
			break
		}
		if !hdr.Has(WalFlagAppended) {
			break
		}
		payloadLen := int(hdr.PayloadLen)
		if payloadLen > MaxPayloadSize {
			return processed, fmt.Errorf("region proxy wal keygen seq=%d: payload too large", hdr.Seq)
		}
		recordLen := int64(HeaderSize + payloadLen)
		if pos+recordLen > writePos {
			break
		}
		if hdr.Has(WalFlagDedupReady) {
			pos += recordLen
			continue
		}
		if len(w.mmap) < int(pos)+HeaderSize+payloadLen {
			return processed, ErrCorrupt
		}
		payload := w.mmap[int(pos)+HeaderSize : int(pos)+HeaderSize+payloadLen]
		factorU, err := derive(hdr.Seq, payload)
		if err != nil {
			return processed, fmt.Errorf("region proxy wal keygen seq=%d: %w", hdr.Seq, err)
		}
		if err := w.markKeyGenReadyLocked(pos, factorU); err != nil {
			return processed, err
		}
		processed++
		pos += recordLen
	}
	return processed, nil
}

func (w *WAL) markKeyGenReadyLocked(headerOff int64, factorU [32]byte) error {
	if len(w.mmap) <= int(headerOff)+flagsOffset {
		return ErrCorrupt
	}
	ctx := context.Background()
	if w.gate != nil {
		if err := w.gate.AcquireAppend(ctx, iogate.TierLow); err != nil {
			return fmt.Errorf("region proxy wal keygen header seq offset=%d: %w", headerOff, err)
		}
	}
	if len(w.mmap) <= int(headerOff)+FactorUOffset+32 {
		if w.gate != nil {
			w.gate.ReleaseAppend(iogate.TierLow)
		}
		return ErrCorrupt
	}
	copy(w.mmap[headerOff+FactorUOffset:headerOff+FactorUOffset+32], factorU[:])
	if len(w.mmap) <= int(headerOff)+flagsOffset {
		if w.gate != nil {
			w.gate.ReleaseAppend(iogate.TierLow)
		}
		return ErrCorrupt
	}
	w.mmap[headerOff+flagsOffset] |= WalFlagDedupReady
	if w.gate != nil {
		w.gate.ReleaseAppend(iogate.TierLow)
	}
	return nil
}

func (w *WAL) WaitKeyGenReady(ctx context.Context, poll time.Duration) error {
	if poll <= 0 {
		poll = time.Millisecond
	}
	ticker := time.NewTicker(poll)
	defer ticker.Stop()
	for {
		if w.KeyGenQueueDepth() == 0 {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}
