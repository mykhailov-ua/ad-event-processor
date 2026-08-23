package edge

import (
	"sync"
	"testing"
)

func TestStore_concurrentApplyDiff(t *testing.T) {
	m := newLPMMap(t)
	store := NewBlocklistStore()

	const workers = 24
	var wg sync.WaitGroup
	wg.Add(workers)
	for i := range workers {
		i := i
		go func() {
			defer wg.Done()
			ip := []string{
				"198.51.100.1",
				"198.51.100.2",
				"203.0.113.10",
				"203.0.113.11",
			}[i%4]
			_, _, _ = store.ApplyDiff(m, nil, []string{ip}, nil, nil)
		}()
	}
	wg.Wait()
	if store.Len() == 0 {
		t.Fatal("expected at least one deny entry after concurrent apply")
	}
}
