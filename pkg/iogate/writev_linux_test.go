//go:build linux

package iogate

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVectoredWrite_coalescesChunks(t *testing.T) {
	f, err := os.CreateTemp("", "iogate-writev-*")
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Remove(f.Name()) })

	chunks := [][]byte{[]byte("hello "), []byte("world")}
	n, err := VectoredWrite(int(f.Fd()), chunks)
	require.NoError(t, err)
	require.Equal(t, 11, n)

	data, err := os.ReadFile(f.Name())
	require.NoError(t, err)
	require.Equal(t, "hello world", string(data))
}

func TestFlushVectored_groupCommitFsync(t *testing.T) {
	f, err := os.CreateTemp("", "iogate-flush-*")
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Remove(f.Name()) })

	g := NewDiskWriteGate(Config{
		AppendCapacity:     4,
		GroupCommitRecords: 2,
	})

	var fsyncs int
	chunks := [][]byte{[]byte("a")}
	for range 3 {
		err := g.FlushVectored(t.Context(), int(f.Fd()), chunks, func() error {
			fsyncs++
			return nil
		})
		require.NoError(t, err)
	}
	require.Equal(t, 1, fsyncs, "two appends per group commit")
}
