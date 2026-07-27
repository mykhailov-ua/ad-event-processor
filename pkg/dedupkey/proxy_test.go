package dedupkey

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProxySourceID_Deterministic(t *testing.T) {
	a := ProxySourceID(1, "node-a")
	b := ProxySourceID(1, "node-a")
	c := ProxySourceID(2, "node-a")
	assert.Equal(t, a, b)
	assert.NotEqual(t, a, c)
}

func TestWriteCanonicalProxyBatchPayload_Deterministic(t *testing.T) {
	payload := []byte(`{"evt":"x"}`)
	var buf [128]byte
	c1 := WriteCanonicalProxyBatchPayload(buf[:0], 42, payload)
	c2 := WriteCanonicalProxyBatchPayload(buf[:0], 42, payload)
	assert.Equal(t, c1, c2)
	assert.Equal(t, FactorU(c1), FactorU(c2))
}

func TestWriteCanonicalProxyBatchPayload_SeqChangesFactorU(t *testing.T) {
	payload := []byte("same")
	var buf [128]byte
	u0 := FactorU(WriteCanonicalProxyBatchPayload(buf[:0], 0, payload))
	u1 := FactorU(WriteCanonicalProxyBatchPayload(buf[:0], 1, payload))
	require.NotEqual(t, uuid.Nil, u0)
	assert.NotEqual(t, u0, u1)
}
