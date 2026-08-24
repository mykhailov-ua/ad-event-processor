package ingestion

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func BenchmarkAttestation_VerifyCookie(b *testing.B) {
	secret := []byte("0123456789abcdef0123456789abcdef")
	cid := uuid.New()
	now := time.Now().Unix()
	token, err := MintAttestationToken(secret, cid, "203.0.113.1", 300, now)
	if err != nil {
		b.Fatal(err)
	}
	h := &AdsPacketHandler{}
	h.ConfigureAttestation([][]byte{secret})
	cookie := []byte("Attestation-Token=" + token)
	b.ReportAllocs()
	for b.Loop() {
		if !h.verifyAttestationCookie(cookie, cid, "203.0.113.1", now+1) {
			b.Fatal("verify failed")
		}
	}
}
