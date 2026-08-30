//go:build !linux

// Non-linux builds lack writev(2); VectoredWrite concatenates chunks then issues one Write syscall.
package iogate

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"time"
)

func VectoredWrite(fd int, chunks [][]byte) (int, error) {
	// os.NewFile duplicates fd for stdlib Write; does not change ownership of caller's fd.
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

// FlushVectored mirrors linux gate semantics (TierHigh, NoteAppend group commit) for cross-platform tests.
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
