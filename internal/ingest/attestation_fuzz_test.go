package ingest

import (
	"testing"

	"github.com/google/uuid"
)

func FuzzAttestationTokenParse(f *testing.F) {
	secret := []byte("0123456789abcdef0123456789abcdef")
	cid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	token, _ := MintAttestationToken(secret, cid, "8.8.8.8", 300, 1_700_000_000)
	f.Add(token)
	f.Add("not-a-token")
	f.Add("")
	f.Fuzz(func(t *testing.T, raw string) {
		if len(raw) > 256 {
			return
		}
		var dst [attestationTokenBinaryLen]byte
		_, _ = decodeAttestationTokenBase64URL([]byte(raw), dst[:])
	})
}

func FuzzAttestationHMAC(f *testing.F) {
	secret := []byte("0123456789abcdef0123456789abcdef")
	cid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	f.Add("8.8.8.8", int32(300), int64(1_700_000_000))
	f.Fuzz(func(t *testing.T, ip string, ttl int32, now int64) {
		if len(ip) > 64 {
			return
		}
		token, err := MintAttestationToken(secret, cid, ip, ttl, now)
		if err != nil {
			return
		}
		h := &AdsPacketHandler{}
		h.ConfigureAttestation([][]byte{secret})
		cookie := []byte("Attestation-Token=" + token)
		_ = h.verifyAttestationCookie(cookie, cid, ip, now)
	})
}
