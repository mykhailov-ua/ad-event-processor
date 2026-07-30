package ivtdetector

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTLSImpersonation(t *testing.T) {
	t.Parallel()

	assert.True(t, IsTLSImpersonating(
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		"37b37375c33a2e6a17b2b6400c436321",
	))
	assert.True(t, IsTLSImpersonating(
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		"python-requests-ja3-fingerprint",
	))
	assert.False(t, IsTLSImpersonating(
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		"chrome-ja3-fingerprint",
	))
	assert.False(t, IsTLSImpersonating(
		"python-requests/2.31.0",
		"37b37375c33a2e6a17b2b6400c436321",
	))
	assert.False(t, IsTLSImpersonating(
		"Mozilla/5.0 Chrome/120.0.0.0",
		"chrome-ja3-fingerprint",
	))
}

func TestIsSuspiciousJA3(t *testing.T) {
	t.Parallel()
	assert.True(t, IsSuspiciousJA3("python-requests-ja3"))
	assert.True(t, IsSuspiciousJA3("37b37375c33a2e6a17b2b6400c436321"))
	assert.False(t, IsSuspiciousJA3("chrome-ja3-fingerprint"))
}
