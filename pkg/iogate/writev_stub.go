//go:build !linux

package iogate

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"time"
)

// VectoredWrite coalesces chunks into one write on non-Linux platforms.
func VectoredWrite(fd int, chunks [][]byte) (int, error) {
	f := os.NewFile(uintptr(fd), "iogate-vectored")
	if f == nil {
		return 0, fmt.Errorf("iogate vectored write: invalid fd %d", fd)
	}
	defer f.Close()
	var buf bytes.Buffer
	for _, chunk := range chunks {
		if _, err := buf.Write(chunk); err != nil {
			return 0, err
		}
	}
	return f.Write(buf.Bytes())
}

// FlushVectored writes chunks and runs fsyncFn when group-commit threshold is met.
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
		return fmt.Errorf("iogate flush vectored write: %w", err)
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
