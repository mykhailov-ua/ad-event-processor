package ingest

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHTTPHeaderValValidSWAR(t *testing.T) {
	valid := []string{
		"",
		"application/json",
		"Mozilla/5.0",
		"203.0.113.10",
		"en-US",
		"\"Chromium\";v=\"120\"",
		"\tspaced\t",
		"abc\tdef",
	}
	for _, s := range valid {
		assert.True(t, httpHeaderValValid([]byte(s)), s)
	}

	invalid := []string{
		"\x00",
		"\r",
		"\n",
		"\x7f",
		"\x01",
		"bad\x0cvalue",
		"ok\xff",
	}
	for _, s := range invalid {
		assert.False(t, httpHeaderValValid([]byte(s)), s)
	}
}

func TestHTTPTokenValidSWAR(t *testing.T) {
	assert.False(t, httpTokenValid(nil))
	assert.False(t, httpTokenValid([]byte("")))
	assert.True(t, httpTokenValid([]byte("Content-Type")))
	assert.True(t, httpTokenValid([]byte("X-Forwarded-For")))
	assert.False(t, httpTokenValid([]byte("bad header")))
	assert.False(t, httpTokenValid([]byte("\x7f")))
}
