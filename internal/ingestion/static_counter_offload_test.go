package ingestion

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// Holdout: P6-3 evaluation rejects external counter stores; hot path stays on Redis Lua + local quanta.
func TestStaticCounterOffload_noExternalCounterStore_holdout(t *testing.T) {
	t.Parallel()
	lower := strings.ToLower(unifiedFilterLua + budgetFastLua)
	require.NotContains(t, lower, "aerospike")
	require.NotContains(t, lower, "dragonfly")
	require.Contains(t, lower, "mget", "Lua must keep atomic MGET debit path")
}
