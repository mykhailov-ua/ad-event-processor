package ingest

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIngressQuotaMap_tryAcquire_perWorkerLimit(t *testing.T) {
	var limits UDPControlLimits
	limits.NumShards = 2
	limits.Limits[0] = 100
	limits.Limits[1] = 200
	m := buildIngressQuotaMap(1, &limits, 4)
	require.NotNil(t, m)

	for range 25 {
		require.True(t, m.TryAcquire(0, 0))
	}
	require.False(t, m.TryAcquire(0, 0))
	require.True(t, m.TryAcquire(0, 1))
}

func TestIngressQuotaMap_epochSwapResetsCounters(t *testing.T) {
	var limits UDPControlLimits
	limits.NumShards = 1
	limits.Limits[0] = 40
	m1 := buildIngressQuotaMap(1, &limits, 2)
	for range 20 {
		require.True(t, m1.TryAcquire(0, 0))
	}
	m2 := buildIngressQuotaMap(2, &limits, 2)
	require.True(t, m2.TryAcquire(0, 0))
}

func TestUDPControl_TryIngress(t *testing.T) {
	c := NewUDPControl(UDPControlConfig{
		Enabled:    true,
		NumShards:  1,
		NumWorkers: 2,
		InitialRPS: 10,
	})
	for range 5 {
		require.True(t, c.TryIngress(0, 0))
	}
	require.False(t, c.TryIngress(0, 0))
	require.True(t, c.TryIngress(0, 1))
}

func TestIngressQuotaMap_race(t *testing.T) {
	var limits UDPControlLimits
	limits.NumShards = 4
	for i := range uint8(4) {
		limits.Limits[i] = 10_000
	}
	m := buildIngressQuotaMap(1, &limits, 8)
	var wg sync.WaitGroup
	for w := range 8 {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			for i := range 2000 {
				_ = m.TryAcquire(i%4, worker)
			}
		}(w)
	}
	wg.Wait()
}
