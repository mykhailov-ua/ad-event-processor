package ingestion

import (
	"testing"
	"time"
)

var linkSignerBenchSink bool

func BenchmarkLinkSigner_Sign(b *testing.B) {
	secret := []byte("bench-link-secret")
	clickID := []byte("bench-click-id")
	expires := time.Now().Add(15 * time.Minute).Unix()
	dst := make([]byte, 0, 256)
	b.ReportAllocs()
	for b.Loop() {
		dst = AppendLinkSignature(dst[:0], secret, clickID, expires)
	}
	linkSignerBenchSink = len(dst) > 0
}

func BenchmarkLinkSigner_Verify(b *testing.B) {
	h := &AdsPacketHandler{}
	h.ConfigureLinkSigning([]byte("bench-link-secret"))
	clickID := []byte("bench-click-id")
	expires := time.Now().Add(15 * time.Minute).Unix()
	loc := AppendLinkSignature([]byte("https://offer.test/lp"), []byte("bench-link-secret"), clickID, expires)
	sig := loc[len(loc)-linkSigHexLen:]
	now := time.Now().Unix()
	b.ReportAllocs()
	var ok bool
	for b.Loop() {
		ok = h.verifyLinkSignature(clickID, sig, expires, now)
	}
	linkSignerBenchSink = ok
}
