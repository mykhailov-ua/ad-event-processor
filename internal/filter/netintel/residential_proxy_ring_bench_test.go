package netintel

import (
	"fmt"
	"sync"
	"testing"
	"unsafe"

	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

var residentialProxyBenchSink bool

func TestResidentialProxyCell_cacheLinePadded_holdout(t *testing.T) {
	size := int(unsafe.Sizeof(residentialProxyCell{}))
	require.Equal(t, 0, size%64, "cell size %d must be cache-line aligned", size)
	require.GreaterOrEqual(t, size, 64*5)
}

func TestResidentialProxyRing_parallelObserveRace(t *testing.T) {
	ring := NewResidentialProxyRing()
	var wg sync.WaitGroup
	for g := range 8 {
		wg.Add(1)
		go func(seed int) {
			defer wg.Done()
			id := uuid.MustParse(fmt.Sprintf("00000000-0000-4000-8000-0000000000%02d", seed))
			hash := domain.CRC32Castagnoli(&id)
			for i := range 2000 {
				_, _ = ring.Observe(hash, i&1 == 0, uint32(i+seed), uint32(i+seed+1), monotonicNano())
			}
		}(g)
	}
	wg.Wait()
}

func BenchmarkResidentialProxy_observe(b *testing.B) {
	ring := NewResidentialProxyRing()
	id := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	hash := domain.CRC32Castagnoli(&id)
	now := monotonicNano()

	b.ReportAllocs()
	var signal bool
	benchN := 0
	for b.Loop() {
		_, signal = ring.Observe(hash, true, uint32(benchN), uint32(benchN+1), now)
		benchN++
	}
	residentialProxyBenchSink = residentialProxyBenchSink || signal
}
