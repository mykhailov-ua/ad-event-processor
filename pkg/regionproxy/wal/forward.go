package wal

import (
	"context"
	"fmt"

	"ad-event-processor/pkg/iogate"
)

func (w *WAL) TryClaimForward(seq uint64) (bool, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	off, hdr, err := w.recordHeaderOffsetLocked(seq)
	if err != nil {
		return false, err
	}
	if !hdr.Has(WalFlagDedupReady) {
		return false, nil
	}
	if hdr.Has(WalFlagForwardClaimed) {
		return false, nil
	}
	if len(w.mmap) <= int(off)+flagsOffset {
		return false, ErrCorrupt
	}
	ctx := context.Background()
	if w.gate != nil {
		if err := w.gate.AcquireAppend(ctx, iogate.TierLow); err != nil {
			return false, fmt.Errorf("region proxy wal forward claim seq=%d: %w", seq, err)
		}
	}
	if len(w.mmap) <= int(off)+flagsOffset {
		if w.gate != nil {
			w.gate.ReleaseAppend(iogate.TierLow)
		}
		return false, ErrCorrupt
	}
	w.mmap[off+flagsOffset] |= WalFlagForwardClaimed
	if w.gate != nil {
		w.gate.ReleaseAppend(iogate.TierLow)
	}
	return true, nil
}

func (w *WAL) UnclaimForward(seq uint64) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	off, hdr, err := w.recordHeaderOffsetLocked(seq)
	if err != nil {
		return err
	}
	if hdr.Has(WalFlagRemoteAcked) {
		return nil
	}
	if !hdr.Has(WalFlagForwardClaimed) {
		return nil
	}
	ctx := context.Background()
	if w.gate != nil {
		if err := w.gate.AcquireAppend(ctx, iogate.TierLow); err != nil {
			return fmt.Errorf("region proxy wal unclaim forward seq=%d: %w", seq, err)
		}
	}
	if len(w.mmap) <= int(off)+flagsOffset {
		if w.gate != nil {
			w.gate.ReleaseAppend(iogate.TierLow)
		}
		return ErrCorrupt
	}
	w.mmap[off+flagsOffset] &^= WalFlagForwardClaimed
	if w.gate != nil {
		w.gate.ReleaseAppend(iogate.TierLow)
	}
	return nil
}

func (w *WAL) MarkRemoteAcked(seq uint64) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	off, hdr, err := w.recordHeaderOffsetLocked(seq)
	if err != nil {
		return err
	}
	if !hdr.Has(WalFlagForwardClaimed) {
		return fmt.Errorf("region proxy wal remote ack seq=%d: forward not claimed", seq)
	}
	if hdr.Has(WalFlagRemoteAcked) {
		return nil
	}
	ctx := context.Background()
	if w.gate != nil {
		if err := w.gate.AcquireAppend(ctx, iogate.TierLow); err != nil {
			return fmt.Errorf("region proxy wal remote ack seq=%d: %w", seq, err)
		}
	}
	if len(w.mmap) <= int(off)+flagsOffset {
		if w.gate != nil {
			w.gate.ReleaseAppend(iogate.TierLow)
		}
		return ErrCorrupt
	}
	w.mmap[off+flagsOffset] |= WalFlagRemoteAcked
	if w.gate != nil {
		w.gate.ReleaseAppend(iogate.TierLow)
	}
	return nil
}

func (w *WAL) ReadRecordPayload(seq uint64) ([]byte, error) {
	_, payload, err := w.ReadRecord(seq)
	return payload, err
}

func (w *WAL) recordHeaderOffsetLocked(seq uint64) (int64, Header, error) {
	pos := int64(0)
	writePos := w.writePos.Load()
	for pos < writePos {
		if len(w.mmap) <= int(pos)+HeaderSize {
			return 0, Header{}, ErrCorrupt
		}
		hdr := readHeaderWithFactor(w.mmap[pos:])
		if hdr.Seq == 0 && hdr.PayloadLen == 0 && hdr.Flags == 0 {
			return 0, Header{}, ErrCorrupt
		}
		payloadLen := int(hdr.PayloadLen)
		recordLen := int64(HeaderSize + payloadLen)
		if pos+recordLen > writePos {
			return 0, Header{}, ErrCorrupt
		}
		if hdr.Seq == seq {
			return pos, hdr, nil
		}
		pos += recordLen
	}
	return 0, Header{}, ErrCorrupt
}
