package opkey

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPool_20ProducersUniqueOpID(t *testing.T) {
	pool := New(nil, Config{NodeID: "node-a", QueueSize: 8192, Watermark: 100000})
	stop := make(chan struct{})
	go func() {
		for {
			select {
			case <-stop:
				return
			default:
				if slot, ok := pool.Dequeue(); ok {
					pool.Release(slot)
				}
			}
		}
	}()
	defer close(stop)

	var wg sync.WaitGroup
	const (
		producers = 20
		perProd   = 500
	)
	var dup atomic.Bool
	seen := make(map[[16]byte]struct{}, producers*perProd)
	var seenMu sync.Mutex

	for range producers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var factor [32]byte
			for range perProd {
				for {
					id, ok := pool.TryEnqueue(1, factor)
					if ok {
						seenMu.Lock()
						if _, exists := seen[id]; exists {
							dup.Store(true)
						} else {
							seen[id] = struct{}{}
						}
						seenMu.Unlock()
						break
					}
					time.Sleep(time.Microsecond)
				}
			}
		}()
	}
	wg.Wait()
	assert.False(t, dup.Load())
	assert.Len(t, seen, producers*perProd)
}

func TestSlot_TryClaimExecutingSingleWinner(t *testing.T) {
	var slot Slot
	slot.Seq = 7
	slot.setDerived()
	require.True(t, slot.TryBook())

	const competitors = 32
	var winners atomic.Int32
	var wg sync.WaitGroup
	for range competitors {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if slot.TryClaimExecuting() {
				winners.Add(1)
			}
		}()
	}
	wg.Wait()
	assert.Equal(t, int32(1), winners.Load())
	assert.True(t, slot.Has(OpKeyFlagExecuting))
}

func TestPool_ShedWhenOverWatermark(t *testing.T) {
	pool := New(nil, Config{NodeID: "node-b", QueueSize: 64, Watermark: 4})
	var factor [32]byte
	for i := range 5 {
		_, ok := pool.TryEnqueue(uint64(i), factor)
		require.True(t, ok)
	}
	_, ok := pool.TryEnqueue(99, factor)
	assert.False(t, ok)
	assert.Equal(t, uint64(1), pool.ShedTotal())
}

func TestPool_FlagProgression(t *testing.T) {
	var slot Slot
	slot.Seq = 1
	slot.setDerived()
	require.True(t, slot.TryBook())
	assert.True(t, slot.Has(OpKeyFlagReplicaBooked))
	assert.False(t, slot.Has(OpKeyFlagExecuting))
	require.True(t, slot.TryClaimExecuting())
	assert.True(t, slot.Has(OpKeyFlagExecuting))
}

func TestSlot_MarkLeaseRenewedRequiresExecuting(t *testing.T) {
	t.Parallel()
	var slot Slot
	require.False(t, slot.MarkLeaseRenewed())

	slot.Seq = 1
	slot.setDerived()
	require.True(t, slot.TryBook())
	require.True(t, slot.TryClaimExecuting())
	require.True(t, slot.MarkLeaseRenewed())
	require.True(t, slot.Has(OpKeyFlagLeaseRenewed))
}
