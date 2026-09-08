package stream

import (
	"testing"

	"ad-event-processor/internal/rtb"

	"github.com/stretchr/testify/require"
)

func TestLocalFcapLedger_tryAcquire_holdout(t *testing.T) {
	ledger := NewLocalFcapLedger()
	prefix := rtb.HashString64("fcap:c:test:u:")
	user := rtb.HashString64("user-a")
	lookup := rtb.FcapLookupKey(prefix, user)
	const limit = 3

	for i := 0; i < int(limit); i++ {
		require.True(t, ledger.TryAcquire(lookup, limit, 3600, 1_700_000_000))
	}
	require.False(t, ledger.TryAcquire(lookup, limit, 3600, 1_700_000_000))
	require.True(t, ledger.WouldExceed(lookup, limit, 3600, 1_700_000_000))

	ledger.Rollback(lookup)
	require.False(t, ledger.WouldExceed(lookup, limit, 3600, 1_700_000_000))
	require.True(t, ledger.TryAcquire(lookup, limit, 3600, 1_700_000_000))
}

func TestLocalFcapLedger_windowReset_holdout(t *testing.T) {
	ledger := NewLocalFcapLedger()
	lookup := rtb.FcapLookupKey(11, 22)
	require.True(t, ledger.TryAcquire(lookup, 1, 60, 100))
	require.False(t, ledger.TryAcquire(lookup, 1, 60, 100))
	require.True(t, ledger.TryAcquire(lookup, 1, 60, 200))
}
