package wal

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"ad-event-processor/pkg/iogate"
)

var (
	ErrCorrupt      = errors.New("wal corrupt")
	ErrSegmentFull  = errors.New("wal segment full")
	ErrEmptyPayload = errors.New("wal empty payload")
	defaultSegSize  = int64(64 * 1024 * 1024)
	walSegmentFile  = "wal.segment"
)

type WAL struct {
	dir      string
	path     string
	gate     *iogate.DiskWriteGate
	file     *os.File
	mmap     []byte
	capacity int64
	writePos atomic.Int64
	nextSeq  atomic.Uint64
	mu       sync.Mutex
}

func Open(dir string, gate *iogate.DiskWriteGate) (*WAL, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("region proxy wal open dir=%s: %w", dir, err)
	}
	path := filepath.Join(dir, walSegmentFile)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, fmt.Errorf("region proxy wal open file=%s: %w", path, err)
	}

	info, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return nil, err
	}
	size := info.Size()
	if size < defaultSegSize {
		if err := f.Truncate(defaultSegSize); err != nil {
			_ = f.Close()
			return nil, err
		}
		size = defaultSegSize
	}

	mmapData, err := syscall.Mmap(int(f.Fd()), 0, int(size), syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	if err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("region proxy wal mmap: %w", err)
	}
	for i := 0; i < len(mmapData); i += 4096 {
		_ = mmapData[i]
	}

	w := &WAL{
		dir:      dir,
		path:     path,
		gate:     gate,
		file:     f,
		mmap:     mmapData,
		capacity: size,
	}
	if err := w.Recover(); err != nil {
		_ = w.Close()
		return nil, err
	}
	return w, nil
}

func (w *WAL) Append(payload []byte) (uint64, error) {
	if len(payload) == 0 {
		return 0, ErrEmptyPayload
	}
	w.mu.Lock()
	defer w.mu.Unlock()

	ctx := context.Background()
	if w.gate != nil {
		if err := w.gate.AcquireAppend(ctx, iogate.TierHigh); err != nil {
			return 0, fmt.Errorf("region proxy wal append seq=%d: %w", w.nextSeq.Load(), err)
		}
	}

	seq, err := w.appendLocked(payload)
	if w.gate != nil {
		w.gate.ReleaseAppend(iogate.TierHigh)
	}
	if err != nil {
		return 0, err
	}

	if w.gate != nil && w.gate.NoteAppend() {
		if err := w.fsyncLocked(); err != nil {
			return seq, fmt.Errorf("region proxy wal append seq=%d: %w", seq, err)
		}
	}
	return seq, nil
}

func (w *WAL) appendLocked(payload []byte) (uint64, error) {
	recordLen := int64(HeaderSize + len(payload))
	pos := w.writePos.Load()
	if pos+recordLen > w.capacity {
		return 0, ErrSegmentFull
	}
	if len(w.mmap) <= int(pos)+HeaderSize {
		return 0, ErrCorrupt
	}

	seq := w.nextSeq.Load()
	hdr := Header{
		Seq:        seq,
		PayloadLen: uint32(len(payload)),
		Flags:      WalFlagAppended,
	}
	copyHeader(w.mmap[pos:], &hdr)

	payloadOff := int(pos) + HeaderSize
	if len(w.mmap) < payloadOff+len(payload) {
		return 0, ErrCorrupt
	}
	copy(w.mmap[payloadOff:payloadOff+len(payload)], payload)

	w.writePos.Store(pos + recordLen)
	w.nextSeq.Store(seq + 1)
	return seq, nil
}

func (w *WAL) AppendBatch(payloads [][]byte) (lastSeq uint64, committed uint32, err error) {
	if len(payloads) == 0 {
		return 0, 0, nil
	}
	w.mu.Lock()
	defer w.mu.Unlock()

	ctx := context.Background()
	if w.gate != nil {
		if err := w.gate.AcquireAppend(ctx, iogate.TierHigh); err != nil {
			return 0, 0, fmt.Errorf("region proxy wal append batch: %w", err)
		}
		defer w.gate.ReleaseAppend(iogate.TierHigh)
	}

	shouldFsync := false
	for _, payload := range payloads {
		if len(payload) == 0 {
			continue
		}
		seq, appendErr := w.appendLocked(payload)
		if appendErr != nil {
			return lastSeq, committed, appendErr
		}
		lastSeq = seq
		committed++
		if w.gate != nil && w.gate.NoteAppend() {
			shouldFsync = true
		}
	}
	if shouldFsync {
		if fsyncErr := w.fsyncLocked(); fsyncErr != nil {
			return lastSeq, committed, fmt.Errorf("region proxy wal append batch: %w", fsyncErr)
		}
	}
	return lastSeq, committed, nil
}

func (w *WAL) fsyncLocked() error {
	ctx := context.Background()
	if w.gate != nil {
		if err := w.gate.AcquireFsync(ctx); err != nil {
			return err
		}
	}
	start := time.Now()
	err := w.file.Sync()
	latency := time.Since(start)
	if w.gate != nil {
		w.gate.ReleaseFsync(latency)
	}
	return err
}

func (w *WAL) Recover() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	pos := int64(0)
	var seq uint64
	for len(w.mmap) > int(pos)+HeaderSize {
		hdr := readHeader(w.mmap[pos:])
		if hdr.Seq == 0 && hdr.PayloadLen == 0 && hdr.Flags == 0 {
			break
		}
		if !hdr.Has(WalFlagAppended) {
			break
		}
		payloadLen := int64(hdr.PayloadLen)
		recordLen := int64(HeaderSize) + payloadLen
		if payloadLen < 0 || pos+recordLen > w.capacity {
			break
		}
		if len(w.mmap) < int(pos)+int(recordLen) {
			break
		}
		seq = hdr.Seq + 1
		pos += recordLen
	}

	w.writePos.Store(pos)
	w.nextSeq.Store(seq)

	if len(w.mmap) > 0 {
		if err := syscall.Munmap(w.mmap); err != nil {
			return fmt.Errorf("region proxy wal recover unmap: %w", err)
		}
		w.mmap = nil
	}
	if err := w.file.Truncate(pos); err != nil {
		return fmt.Errorf("region proxy wal recover truncate tail: %w", err)
	}
	if err := w.file.Truncate(w.capacity); err != nil {
		return fmt.Errorf("region proxy wal recover truncate: %w", err)
	}
	mmapData, err := syscall.Mmap(int(w.file.Fd()), 0, int(w.capacity), syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	if err != nil {
		return fmt.Errorf("region proxy wal recover mmap: %w", err)
	}
	for i := 0; i < len(mmapData); i += 4096 {
		_ = mmapData[i]
	}
	w.mmap = mmapData
	return nil
}

func (w *WAL) NextSeq() uint64 {
	return w.nextSeq.Load()
}

func (w *WAL) WritePos() int64 {
	return w.writePos.Load()
}

func (w *WAL) Gate() *iogate.DiskWriteGate {
	return w.gate
}

func (w *WAL) ReadRecord(seq uint64) (Header, []byte, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	pos := int64(0)
	for {
		if len(w.mmap) <= int(pos)+HeaderSize {
			return Header{}, nil, ErrCorrupt
		}
		hdr := readHeader(w.mmap[pos:])
		if hdr.Seq == 0 && hdr.PayloadLen == 0 && hdr.Flags == 0 {
			return Header{}, nil, ErrCorrupt
		}
		payloadLen := int(hdr.PayloadLen)
		recordLen := int64(HeaderSize + payloadLen)
		if pos+recordLen > w.writePos.Load() {
			return Header{}, nil, ErrCorrupt
		}
		if hdr.Seq == seq {
			if len(w.mmap) < int(pos)+HeaderSize+payloadLen {
				return Header{}, nil, ErrCorrupt
			}
			payload := make([]byte, payloadLen)
			copy(payload, w.mmap[int(pos)+HeaderSize:int(pos)+HeaderSize+payloadLen])
			hdr = readHeaderWithFactor(w.mmap[pos:])
			return hdr, payload, nil
		}
		pos += recordLen
	}
}

func (w *WAL) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	var err error
	if len(w.mmap) > 0 {
		if munmapErr := syscall.Munmap(w.mmap); munmapErr != nil {
			err = munmapErr
		}
		w.mmap = nil
	}
	if w.file != nil {
		if closeErr := w.file.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
		w.file = nil
	}
	return err
}

func copyHeader(dst []byte, hdr *Header) {
	if len(dst) < HeaderSize {
		return
	}
	binary.LittleEndian.PutUint64(dst[0:8], hdr.Seq)
	binary.LittleEndian.PutUint32(dst[8:12], hdr.PayloadLen)
	dst[flagsOffset] = hdr.Flags
}

func readHeader(src []byte) Header {
	if len(src) < HeaderSize {
		return Header{}
	}
	return Header{
		Seq:        binary.LittleEndian.Uint64(src[0:8]),
		PayloadLen: binary.LittleEndian.Uint32(src[8:12]),
		Flags:      src[flagsOffset],
	}
}

func readHeaderWithFactor(src []byte) Header {
	hdr := readHeader(src)
	if len(src) >= FactorUOffset+32 {
		copy(hdr.FactorU[:], src[FactorUOffset:FactorUOffset+32])
	}
	return hdr
}

func (w *WAL) ReadRawMessages(startSeq uint64, maxBytes uint32) (data []byte, bufPtr *[]byte, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	pos := int64(0)
	writePos := w.writePos.Load()
	var out []byte

	for len(w.mmap) > int(pos)+HeaderSize {
		hdr := readHeader(w.mmap[pos:])
		if hdr.Seq == 0 && hdr.PayloadLen == 0 && hdr.Flags == 0 {
			break
		}
		if !hdr.Has(WalFlagAppended) {
			break
		}
		payloadLen := int(hdr.PayloadLen)
		recordLen := int64(HeaderSize + payloadLen)
		if pos+recordLen > writePos {
			break
		}
		if hdr.Seq < startSeq {
			pos += recordLen
			continue
		}
		wireLen := 12 + payloadLen
		if maxBytes > 0 && uint32(len(out)+wireLen) > maxBytes {
			break
		}
		if len(w.mmap) < int(pos)+HeaderSize+payloadLen {
			return nil, nil, ErrCorrupt
		}
		if out == nil {
			pool := fetchBufPool.Get().(*[]byte)
			bufPtr = pool
			out = (*pool)[:0]
		}
		need := len(out) + wireLen
		if need > cap(*bufPtr) {
			return nil, nil, ErrCorrupt
		}
		off := len(out)
		out = (*bufPtr)[:need]
		binary.BigEndian.PutUint32(out[off:off+4], uint32(8+payloadLen))
		binary.BigEndian.PutUint64(out[off+4:off+12], hdr.Seq)
		copy(out[off+12:off+12+payloadLen], w.mmap[int(pos)+HeaderSize:int(pos)+HeaderSize+payloadLen])
		pos += recordLen
	}

	if len(out) == 0 {
		return nil, nil, io.EOF
	}
	return out, bufPtr, nil
}

var fetchBufPool = sync.Pool{
	New: func() any {
		b := make([]byte, 1024*1024)
		return &b
	},
}
