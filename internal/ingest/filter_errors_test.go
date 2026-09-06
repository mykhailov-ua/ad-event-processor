package ingest

import (
	"context"
	"net/http"
	"testing"

	"ad-event-processor/internal/database"

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

	kind, ok = classifyFilterErr(ErrInfraNetwork)
	require.True(t, ok)
	assert.Equal(t, filterRejectInfra, kind)
	assert.Equal(t, http.StatusServiceUnavailable, filterRejectSpecs[kind].status)
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

func TestClassifyFilterErr_infraFailClosed503_holdout(t *testing.T) {
	infraCases := []struct {
		err        error
		wantKind   filterRejectKind
		wantStatus int
	}{
		{ErrRegistryStale, filterRejectRegistryStale, http.StatusServiceUnavailable},
		{ErrShardUnavailable, filterRejectShardUnavailable, http.StatusServiceUnavailable},
		{ErrGeoLookupFailed, filterRejectInfra, http.StatusServiceUnavailable},
		{database.ErrRedisCircuitOpen, filterRejectInfra, http.StatusServiceUnavailable},
	}
	for _, tc := range infraCases {
		kind, ok := classifyFilterErr(tc.err)
		require.True(t, ok, tc.err)
		assert.Equal(t, tc.wantKind, kind)
		assert.Equal(t, tc.wantStatus, filterRejectSpecs[kind].status)
		assert.NotEqual(t, http.StatusNotFound, filterRejectSpecs[kind].status)
	}
}
