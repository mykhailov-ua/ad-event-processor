package ingestion

import (
	"context"
	"net/http"
	"testing"

	"espx/internal/database"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClassifyFilterErr_timeoutVsInfra(t *testing.T) {
	kind, ok := classifyFilterErr(ErrFilterTimeout)
	require.True(t, ok)
	assert.Equal(t, filterRejectTimeout, kind)
	assert.Equal(t, http.StatusGatewayTimeout, filterRejectSpecs[kind].status)

	kind, ok = classifyFilterErr(context.DeadlineExceeded)
	require.True(t, ok)
	assert.Equal(t, filterRejectInfra, kind)
	assert.Equal(t, http.StatusServiceUnavailable, filterRejectSpecs[kind].status)

	kind, ok = classifyFilterErr(database.ErrRedisCircuitOpen)
	require.True(t, ok)
	assert.Equal(t, filterRejectInfra, kind)
	assert.NotEqual(t, http.StatusGatewayTimeout, filterRejectSpecs[kind].status)
}

func TestClassifyFilterErr_noAccidental504(t *testing.T) {
	cases := []error{
		ErrEmergencyBreakerActive,
		ErrRateLimitExceeded,
		ErrBudgetExhausted,
		ErrCampaignNotFound,
		database.ErrRedisCircuitOpen,
		ErrShardUnavailable,
	}
	for _, err := range cases {
		kind, ok := classifyFilterErr(err)
		require.True(t, ok, err)
		assert.NotEqual(t, http.StatusGatewayTimeout, filterRejectSpecs[kind].status, "err=%v kind=%d", err, kind)
	}
}
