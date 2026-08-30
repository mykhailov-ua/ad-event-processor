package ingest

import (
	"sync"
	"testing"
	"unsafe"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestIPv6RotationCell_cacheLinePadded_holdout(t *testing.T) {
	size := int(unsafe.Sizeof(ipv6RotationCell{}))
	require.Equal(t, 0, size%64, "cell size %d must be cache-line aligned", size)
	require.GreaterOrEqual(t, size, 64*2)
}

func TestIPv6RotationTable_parallelObserveRace(t *testing.T) {
	table := NewIPv6RotationTable()
	table.SetMode("shadow")
	table.SetPolicy(uint64(defaultIPv6RotationWindow), defaultIPv6RotationThresh)
	cid := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	campaignHash := crc32Castagnoli(&cid)
	v6Hi := uint64(0x20010db885a30000)

	var wg sync.WaitGroup
	for g := range 8 {
		wg.Add(1)
		go func(seed int) {
			defer wg.Done()
			now := monotonicNano()
			for i := range 2000 {
				_, _ = table.Observe(campaignHash, v6Hi, uint64(i+seed*1000), now)
			}
		}(g)
	}
	wg.Wait()
}
