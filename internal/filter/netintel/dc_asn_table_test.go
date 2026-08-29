package netintel

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseDCASNFeed(t *testing.T) {
	asns := parseDCASNFeed([]byte("# dc\nAS16509\n14618\n\n"))
	require.Len(t, asns, 2)
	_, ok := asns[16509]
	assert.True(t, ok)
	_, ok = asns[14618]
	assert.True(t, ok)
}
