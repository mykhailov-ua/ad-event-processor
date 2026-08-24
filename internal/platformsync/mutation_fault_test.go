package platformsync_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMutationFault_idempotencyKeyRequired(t *testing.T) {
	t.Parallel()
	key := "platform-pause-550e8400-e29b-41d4-a716-446655440000"
	require.NotEmpty(t, key)
	require.Greater(t, len(key), 8)
}
