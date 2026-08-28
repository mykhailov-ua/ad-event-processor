package fraudadmin

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestInvalidatePolicyPresetCache(t *testing.T) {
	policyPresetCache.mu.Lock()
	policyPresetCache.loadedAt = time.Now()
	policyPresetCache.rows = []policyPresetRow{{name: "balanced"}}
	policyPresetCache.mu.Unlock()

	InvalidatePolicyPresetCache()

	policyPresetCache.mu.RLock()
	defer policyPresetCache.mu.RUnlock()
	require.True(t, policyPresetCache.loadedAt.IsZero())
	require.Nil(t, policyPresetCache.rows)
}
