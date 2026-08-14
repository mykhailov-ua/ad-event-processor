package iogate

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGroupCommit_FsyncReduction70Percent(t *testing.T) {
	const appends = 1000
	g := NewDiskWriteGate(Config{
		AppendCapacity:     DefaultAppendCapacity,
		GroupCommitRecords: DefaultGroupCommitRecords,
	})

	var fsyncs int
	for range appends {
		if g.NoteAppend() {
			fsyncs++
		}
	}

	reduction := float64(appends-fsyncs) / float64(appends)
	require.GreaterOrEqual(t, reduction, 0.70,
		"group commit fsync reduction = %.1f%% (%d fsyncs / %d appends)", reduction*100, fsyncs, appends)
}
