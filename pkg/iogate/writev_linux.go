//go:build linux

package iogate

import (
	"context"
	"fmt"
	"syscall"
	"time"
	"unsafe"
)

// VectoredWrite issues one writev syscall for multiple chunks (GAP-DB-01 write coalescing).
func VectoredWrite(fd int, chunks [][]byte) (int, error) {
	if len(chunks) == 0 {
		return 0, nil
	}
	if len(chunks) == 1 {
		return syscall.Write(fd, chunks[0])
	}
	iovecs := make([]syscall.Iovec, len(chunks))
	for i, chunk := range chunks {
		if len(chunk) == 0 {
			continue
		}
		iovecs[i].Base = &chunk[0]
		iovecs[i].SetLen(len(chunk))
	}
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

// FlushVectored writes chunks with writev and runs fsyncFn when group-commit threshold is met.
func (g *DiskWriteGate) FlushVectored(ctx context.Context, fd int, chunks [][]byte, fsyncFn func() error) error {
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
	if !g.NoteAppend() {
		return nil
	}
	if err := g.AcquireFsync(ctx); err != nil {
		return fmt.Errorf("iogate flush vectored fsync: %w", err)
	}
	fsyncStart := time.Now()
	err := fsyncFn()
	g.ReleaseFsync(time.Since(fsyncStart))
	return err
}
