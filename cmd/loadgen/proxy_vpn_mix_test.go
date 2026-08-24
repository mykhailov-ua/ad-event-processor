package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProxyVPNClientIP_rotates(t *testing.T) {
	a := proxyVPNClientIP(0)
	b := proxyVPNClientIP(1)
	assert.NotEmpty(t, a)
	assert.NotEmpty(t, b)
	assert.NotEqual(t, a, b)
}

func TestMix_carveProxyVPNAndFlowRoute(t *testing.T) {
	mix := defaultMix("business", 50, 20)
	for _, pct := range []int{3, 2} {
		if pct > mix.pctValid {
			pct = mix.pctValid
		}
		mix.pctValid -= pct
		mix.pctProxyVPN += pct
	}
	total := mix.pctOpenRTB + mix.pctTelegram + mix.pctValid + mix.pctFraud +
		mix.pctInvalid + mix.pctDDoS + mix.pctClickProxy + mix.pctProxyVPN + mix.pctFlowRoute
	assert.LessOrEqual(t, total, 100)
}
