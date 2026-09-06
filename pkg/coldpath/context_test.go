package coldpath

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestBoundedContext_respectsParentDeadline_holdout(t *testing.T) {
	parent, parentCancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer parentCancel()

	ctx, cancel := BoundedContext(parent, time.Minute)
	defer cancel()

	deadline, ok := ctx.Deadline()
	require.True(t, ok)
	require.Less(t, time.Until(deadline), time.Minute)
	require.Greater(t, time.Until(deadline), time.Duration(0))
}
