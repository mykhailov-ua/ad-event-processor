package httpingress

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHTTP1ClientIPNeedsRealIPFallback(t *testing.T) {
	assert.True(t, http1ClientIPNeedsRealIPFallback([]byte("192.168.1.50")))
	assert.True(t, http1ClientIPNeedsRealIPFallback([]byte("192.168.1.50, 10.0.0.99")))
	assert.False(t, http1ClientIPNeedsRealIPFallback([]byte("203.0.113.10")))
}
