package controlplane

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"espx/pkg/dedupkey"
)

func TestOperation_RelayDeliveryOpIDDeterministic(t *testing.T) {
	t.Parallel()
	a := RelayDeliveryOpID(3, 42)
	b := RelayDeliveryOpID(3, 42)
	c := RelayDeliveryOpID(3, 43)
	assert.Equal(t, a, b)
	assert.NotEqual(t, a, c)
}

func TestOperation_ProxyBatchOpIDUsesProvided(t *testing.T) {
	t.Parallel()
	op := uuid.New()
	assert.Equal(t, op, ProxyBatchOpID(1, "node-a", 99, op))
}

func TestOperation_ProxyBatchOpIDFallback(t *testing.T) {
	t.Parallel()
	a := ProxyBatchOpID(2, "node-b", 7, uuid.Nil)
	b := ProxyBatchOpID(2, "node-b", 7, uuid.Nil)
	assert.Equal(t, a, b)
	assert.NotEqual(t, uuid.Nil, a)
}

func TestOperation_LeaseDedupScopeRoundTrip(t *testing.T) {
	t.Parallel()
	scope := dedupkey.Scope{
		RegionID:    uuid.New(),
		SourceID:    uuid.New(),
		SourceEpoch: 3,
		SeqStart:    10,
		SeqEnd:      20,
	}
	raw, err := EncodeLeaseDedupScope(scope, 2)
	require.NoError(t, err)
	got, attempt, err := DecodeLeaseDedupScope(raw)
	require.NoError(t, err)
	assert.Equal(t, scope, got)
	assert.Equal(t, int32(2), attempt)
}

func TestOperation_ProxyBatchOpIDFromBytes(t *testing.T) {
	t.Parallel()
	var zero [16]byte
	assert.Equal(t, uuid.Nil, ProxyBatchOpIDFromBytes(zero))
	var raw [16]byte
	id := uuid.New()
	copy(raw[:], id[:])
	assert.Equal(t, id, ProxyBatchOpIDFromBytes(raw))
}

func TestOperation_DomainMapped(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "operation", FileDomain("operation_lease_paths.go"))
}
