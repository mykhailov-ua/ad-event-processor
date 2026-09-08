package campaign

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestBulkCloneIdempotencyKey_holdout(t *testing.T) {
	sourceID := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	key := BulkCloneIdempotencyKey("bulk-key", sourceID)
	require.Equal(t, "bulk-key:00000000-0000-4000-8000-000000000001", key)
}
