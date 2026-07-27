package iogate

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiskWriteGate_ConcurrentFsyncSerializes(t *testing.T) {
	g := NewDiskWriteGate(Config{AppendCapacity: 32})

	const workers = 32
	var maxInFlight atomic.Int32
	var wg sync.WaitGroup
	start := make(chan struct{})

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			require.NoError(t, g.AcquireFsync(ctx))
			cur := int32(g.FsyncInFlight())
			for {
				prev := maxInFlight.Load()
				if cur <= prev || maxInFlight.CompareAndSwap(prev, cur) {
					break
				}
			}
			time.Sleep(2 * time.Millisecond)
			g.ReleaseFsync(time.Millisecond)
		}()
	}
	close(start)
	wg.Wait()

	assert.Equal(t, 1, int(maxInFlight.Load()))
	assert.Equal(t, 0, g.FsyncInFlight())
}

func TestDiskWriteGate_DegradedShedsTierLowBlocksTierHigh(t *testing.T) {
	g := NewDiskWriteGate(Config{AppendCapacity: 1})
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	require.NoError(t, g.AcquireAppend(ctx, TierHigh))

	g.SetDegraded(true)

	errLow := g.AcquireAppend(context.Background(), TierLow)
	require.Error(t, errLow)
	assert.ErrorIs(t, errLow, ErrShed)

	ctxHigh, cancelHigh := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancelHigh()
	errHigh := g.AcquireAppend(ctxHigh, TierHigh)
	require.Error(t, errHigh)
	assert.ErrorIs(t, errHigh, context.DeadlineExceeded)

	g.ReleaseAppend(TierHigh)

	require.NoError(t, g.AcquireAppend(ctx, TierHigh))
	g.ReleaseAppend(TierHigh)
}

func TestDiskWriteGate_EMACrossesBudgetWithinTwoSamples(t *testing.T) {
	budget := 50 * time.Millisecond
	g := NewDiskWriteGate(Config{
		AppendCapacity:    4,
		DiskLatencyBudget: budget,
	})
	require.False(t, g.Degraded())

	g.recordFsyncLatency(2 * budget)
	require.True(t, g.Degraded(), "first sample above budget should flip degraded")

	g2 := NewDiskWriteGate(Config{
		AppendCapacity:    4,
		DiskLatencyBudget: budget,
	})
	g2.recordFsyncLatency(budget * 49 / 50)
	require.False(t, g2.Degraded())
	g2.recordFsyncLatency(100 * time.Millisecond)
	require.True(t, g2.Degraded(), "second sample should cross budget within two updates")
}

func TestDiskWriteGate_GroupCommitThreshold(t *testing.T) {
	g := NewDiskWriteGate(Config{
		AppendCapacity:     4,
		GroupCommitRecords: 4,
	})
	for i := 0; i < 3; i++ {
		assert.False(t, g.NoteAppend())
	}
	assert.True(t, g.NoteAppend())
}

func TestDiskWriteGate_GroupCommitInterval(t *testing.T) {
	g := NewDiskWriteGate(Config{
		AppendCapacity:      4,
		GroupCommitRecords:  64,
		GroupCommitInterval: 10 * time.Millisecond,
	})
	assert.False(t, g.NoteAppend())
	time.Sleep(15 * time.Millisecond)
	assert.True(t, g.NoteAppend())
}

func TestDiskWriteGate_DiskWritableShedsTierLow(t *testing.T) {
	writable := atomic.Bool{}
	writable.Store(false)
	g := NewDiskWriteGate(Config{
		AppendCapacity: 4,
		DiskWritable:   writable.Load,
	})
	err := g.AcquireAppend(context.Background(), TierLow)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrShed)
	assert.True(t, g.Degraded())
}

func TestDiskWriteGate_NilGateNoops(t *testing.T) {
	var g *DiskWriteGate
	ctx := context.Background()
	require.NoError(t, g.AcquireAppend(ctx, TierHigh))
	g.ReleaseAppend(TierHigh)
	require.NoError(t, g.AcquireFsync(ctx))
	g.ReleaseFsync(time.Millisecond)
}

func TestDiskWriteGate_AcquireAppendErrorFormat(t *testing.T) {
	g := NewDiskWriteGate(Config{AppendCapacity: 1})
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	require.NoError(t, g.AcquireAppend(ctx, TierHigh))

	g.SetDegraded(true)
	err := g.AcquireAppend(context.Background(), TierLow)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "disk gate acquire tier=1")
	assert.True(t, errors.Is(err, ErrShed))

	g.ReleaseAppend(TierHigh)
}
