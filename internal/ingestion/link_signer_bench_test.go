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
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
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
	b.ResetTimer()
	var ok bool
	for i := 0; i < b.N; i++ {
		ok = h.verifyLinkSignature(clickID, sig, expires, now)
	}
	linkSignerBenchSink = ok
}
