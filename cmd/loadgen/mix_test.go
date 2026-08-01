package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultMix_includesOpenRTB26(t *testing.T) {
	for _, mode := range []string{"smoke", "full", "business"} {
		mix := defaultMix(mode, 0, 0)
		assert.Greater(t, mix.pctOpenRTB, 0, "mode=%s", mode)
		total := mix.pctOpenRTB + mix.pctValid + mix.pctFraud + mix.pctInvalid + mix.pctDDoS
		assert.LessOrEqual(t, total, 100, "mode=%s", mode)
	}
}

func TestOpenRTBBidBody_shape(t *testing.T) {
	body := openrtbBidBody(42)
	s := string(body)
	assert.Contains(t, s, `"id":"load-42"`)
	assert.Contains(t, s, `"imp":[`)
	assert.Contains(t, s, `"site":`)
	assert.Contains(t, s, `"device":`)
}
