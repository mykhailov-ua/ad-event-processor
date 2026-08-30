//go:build linux

// Linux-only writev(2) scatter path; non-linux builds use writev_stub.go (buffered Write).
package iogate

import (
	"context"
	"fmt"
	"syscall"
	"time"
	"unsafe"
)

func VectoredWrite(fd int, chunks [][]byte) (int, error) {
	if len(chunks) == 0 {
		return 0, nil
	}
	if len(chunks) == 1 {
		// Single-chunk fast path avoids iovecs alloc; same fd write syscall boundary as writev.
		return syscall.Write(fd, chunks[0])
	}
	// iovecs slice alloc per call; acceptable on WAL append lane, not tracker /track.
	iovecs := make([]syscall.Iovec, len(chunks))
	for i, chunk := range chunks {
		if len(chunk) == 0 {
			continue
		}
		iovecs[i].Base = &chunk[0]
		iovecs[i].SetLen(len(chunk))
	}
	// writev(2): scatter single syscall for len(chunks) iovecs.
	n, _, errno := syscall.Syscall(
		syscall.SYS_WRITEV,
		uintptr(fd),
		uintptr(unsafe.Pointer(&iovecs[0])),
		uintptr(len(iovecs)),
	)
	if errno != 0 {
		return int(n), errno
	}
	return int(n), nil
}

// FlushVectored: TierHigh append slot, writev(2), then optional group-commit fsync via fsyncFn (caller supplies Sync/fdatasync).
func (g *DiskWriteGate) FlushVectored(ctx context.Context, fd int, chunks [][]byte, fsyncFn func() error) error {
	// nil gate: writev every call; fsync when fsyncFn set (no group commit).
	if g == nil {
		if _, err := VectoredWrite(fd, chunks); err != nil {
			return err
		}
		if fsyncFn != nil {
			return fsyncFn()
		}
		return nil
	}
	if err := g.AcquireAppend(ctx, TierHigh); err != nil {
		return fmt.Errorf("iogate flush vectored: %w", err)
	}
	defer g.ReleaseAppend(TierHigh)

	if _, err := VectoredWrite(fd, chunks); err != nil {
		return fmt.Errorf("iogate flush vectored writev: %w", err)
	}
	if !g.NoteAppend() { // below group-commit thresholds: skip fsync
		return nil
	}
	if err := g.AcquireFsync(ctx); err != nil {
		return fmt.Errorf("iogate flush vectored fsync: %w", err)
	}
	fsyncStart := time.Now()
	err := fsyncFn() // syscall boundary: typically (*os.File).Sync or unix.Fdatasync
	g.ReleaseFsync(time.Since(fsyncStart))
	return err
}
