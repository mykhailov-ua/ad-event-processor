package ingestion

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLinkSigner_roundTrip(t *testing.T) {
	secret := []byte("unit-test-secret")
	clickID := []byte("click-xyz")
	expires := time.Now().Add(15 * time.Minute).Unix()
	loc := AppendLinkSignature([]byte("https://offer.test/lp?x=1"), secret, clickID, expires)
	require.Contains(t, string(loc), "_sig=")

	sigStart := len(loc) - linkSigHexLen
	sig := loc[sigStart:]
	require.True(t, VerifyLinkSignature(secret, clickID, sig, expires, time.Now().Unix()))
}

func TestLinkSigner_tamperedSigRejected(t *testing.T) {
	secret := []byte("unit-test-secret")
	clickID := []byte("click-xyz")
	expires := time.Now().Add(15 * time.Minute).Unix()
	sig := []byte("abcdef0123456789abcdef0123456789")
	assert.False(t, VerifyLinkSignature(secret, clickID, sig, expires, time.Now().Unix()))
}

func TestLinkSigner_expiredRejected(t *testing.T) {
	secret := []byte("unit-test-secret")
	clickID := []byte("click-xyz")
	expires := time.Now().Add(-1 * time.Minute).Unix()
	loc := AppendLinkSignature([]byte("https://offer.test/lp"), secret, clickID, expires)
	sig := loc[len(loc)-linkSigHexLen:]
	assert.False(t, VerifyLinkSignature(secret, clickID, sig, expires, time.Now().Unix()))
}

func TestLinkSigner_VerifyZeroAlloc(t *testing.T) {
	h := &AdsPacketHandler{}
	h.ConfigureLinkSigning([]byte("pool-secret"))
	clickID := []byte("click-pool")
	expires := time.Now().Add(10 * time.Minute).Unix()
	loc := AppendLinkSignature([]byte("https://offer.test/lp"), []byte("pool-secret"), clickID, expires)
	sig := loc[len(loc)-linkSigHexLen:]
	now := time.Now().Unix()
	allocs := testing.AllocsPerRun(1000, func() {
		if !h.verifyLinkSignature(clickID, sig, expires, now) {
			t.Fatal("verify failed")
		}
	})
	if allocs != 0 {
		t.Fatalf("verify allocs per run = %v, want 0", allocs)
	}
}
