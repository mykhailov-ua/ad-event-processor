package fraud

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResidentialIntelFeedLine_ipv4AndV6(t *testing.T) {
	line, ok := residentialIntelFeedLine("203.0.113.88")
	require.True(t, ok)
	assert.Equal(t, "203.0.113.88/32 0 vpn", line)

	line6, ok := residentialIntelFeedLine("2001:db8::1")
	require.True(t, ok)
	assert.True(t, strings.HasPrefix(line6, "2001:db8::1/128"))
}

func TestResidentialIntelResult_holdoutWithoutFarmHeuristic(t *testing.T) {
	assert.False(t, ResidentialIntelResult{Proxy: true}.IsResidentialProxyFarm())
}
