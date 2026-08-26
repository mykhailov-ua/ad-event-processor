package edge

import (
	"runtime"
	"sync"
	"testing"

	"github.com/cilium/ebpf"
	"github.com/stretchr/testify/require"
)

var bpfTestMu sync.Mutex

func withBPFTestLock(t *testing.T) {
	t.Helper()
	bpfTestMu.Lock()
	t.Cleanup(func() { bpfTestMu.Unlock() })
}

func resetGlobalSynMap(t *testing.T, m *ebpf.Map) {
	t.Helper()
	if m == nil {
		return
	}
	key := uint32(0)
	zeros := make([]EdgeSynState, runtime.NumCPU())
	require.NoError(t, m.Update(&key, zeros, ebpf.UpdateAny))
}
