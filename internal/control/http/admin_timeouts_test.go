package http

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestManagementWriteTimeout_longRoute_holdout(t *testing.T) {
	long := map[string]time.Duration{
		"POST /api/v1/cost-sync/run": 110 * time.Second,
	}
	got := ManagementWriteTimeout(10_000, long)
	require.Equal(t, 110*time.Second, got)
}

func TestManagementWriteTimeout_defaultWhenNoLongRoutes(t *testing.T) {
	got := ManagementWriteTimeout(10_000, nil)
	require.Equal(t, 10*time.Second, got)
}
