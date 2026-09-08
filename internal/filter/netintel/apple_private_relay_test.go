package netintel

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrivateRelay_holdoutRelayASNAppleUAExempt(t *testing.T) {
	table := NewApplePrivateRelayTable(nil)
	require.True(t, table.ExemptFromDCASN(13335, "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X)"))
	assert.False(t, table.ExemptFromDCASN(13335, "Mozilla/5.0 (Windows NT 10.0; Win64; x64)"))
}

func TestPrivateRelay_holdoutNonRelayASN(t *testing.T) {
	table := NewApplePrivateRelayTable(nil)
	assert.False(t, table.ExemptFromDCASN(16509, "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X)"))
}
