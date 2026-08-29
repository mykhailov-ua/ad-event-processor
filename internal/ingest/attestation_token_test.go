package ingest

import (
	"net"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestMintAttestationToken_roundTrip(t *testing.T) {
	secret := []byte("0123456789abcdef0123456789abcdef")
	cid := uuid.New()
	now := time.Now().Unix()
	token, err := MintAttestationToken(secret, cid, "203.0.113.9", 300, now)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	h := &AdsPacketHandler{}
	h.ConfigureAttestation([][]byte{secret})
	cookie := []byte("Attestation-Token=" + token)
	require.True(t, h.verifyAttestationCookie(cookie, cid, "203.0.113.9", now+1))
}

func TestMintAttestationToken_expiredRejected(t *testing.T) {
	secret := []byte("0123456789abcdef0123456789abcdef")
	cid := uuid.New()
	now := int64(1_700_000_000)
	token, err := MintAttestationToken(secret, cid, "8.8.8.8", 60, now)
	require.NoError(t, err)
	h := &AdsPacketHandler{}
	h.ConfigureAttestation([][]byte{secret})
	cookie := []byte("Attestation-Token=" + token)
	require.False(t, h.verifyAttestationCookie(cookie, cid, "8.8.8.8", now+120))
}

func TestMintAttestationToken_wrongIPRejected(t *testing.T) {
	secret := []byte("0123456789abcdef0123456789abcdef")
	cid := uuid.New()
	now := time.Now().Unix()
	token, err := MintAttestationToken(secret, cid, "8.8.8.8", 300, now)
	require.NoError(t, err)
	h := &AdsPacketHandler{}
	h.ConfigureAttestation([][]byte{secret})
	cookie := []byte("Attestation-Token=" + token)
	require.False(t, h.verifyAttestationCookie(cookie, cid, "8.8.4.4", now+1))
}

func TestMintAttestationToken_wrongCampaignRejected(t *testing.T) {
	secret := []byte("0123456789abcdef0123456789abcdef")
	cid := uuid.New()
	now := time.Now().Unix()
	token, err := MintAttestationToken(secret, cid, "8.8.8.8", 300, now)
	require.NoError(t, err)
	h := &AdsPacketHandler{}
	h.ConfigureAttestation([][]byte{secret})
	cookie := []byte("Attestation-Token=" + token)
	require.False(t, h.verifyAttestationCookie(cookie, uuid.New(), "8.8.8.8", now+1))
}

func TestMintAttestationToken_prevSecretRotation(t *testing.T) {
	current := []byte("0123456789abcdef0123456789abcdef")
	prev := []byte("fedcba9876543210fedcba9876543210")
	cid := uuid.New()
	now := time.Now().Unix()
	token, err := MintAttestationToken(prev, cid, "1.2.3.4", 300, now)
	require.NoError(t, err)
	h := &AdsPacketHandler{}
	h.ConfigureAttestation([][]byte{current, prev})
	cookie := []byte("Attestation-Token=" + token)
	require.True(t, h.verifyAttestationCookie(cookie, cid, "1.2.3.4", now+1))
}

func TestAttestation_VerifyCookie_zeroAlloc(t *testing.T) {
	secret := []byte("0123456789abcdef0123456789abcdef")
	cid := uuid.New()
	now := time.Now().Unix()
	token, err := MintAttestationToken(secret, cid, "203.0.113.1", 300, now)
	require.NoError(t, err)
	h := &AdsPacketHandler{}
	h.ConfigureAttestation([][]byte{secret})
	cookie := []byte("Attestation-Token=" + token)
	allocs := testing.AllocsPerRun(100, func() {
		_ = h.verifyAttestationCookie(cookie, cid, "203.0.113.1", now+1)
	})
	require.LessOrEqual(t, allocs, float64(1), "verify path should not allocate beyond base64 decode")
}

func TestDecodeAttestationTokenBase64URL_rejectsMalformedLength(t *testing.T) {
	var dst [attestationTokenBinaryLen]byte
	long := make([]byte, 88)
	for i := range long {
		long[i] = '0'
	}
	_, ok := decodeAttestationTokenBase64URL(long, dst[:])
	require.False(t, ok)
	_, ok = decodeAttestationTokenBase64URL([]byte("short"), dst[:])
	require.False(t, ok)
}

func TestExtractAttestationCookie_multipleCookies(t *testing.T) {
	hdr := []byte("session=abc; Attestation-Token=token123; other=1")
	got := extractAttestationCookie(hdr)
	require.Equal(t, []byte("token123"), got)
}

func TestEncodeAttestationIPPrefix_ipv6(t *testing.T) {
	var dst [16]byte
	require.True(t, encodeAttestationIPPrefix("2001:db8::1", dst[:]))
	ip := net.ParseIP("2001:db8::2")
	require.True(t, attestationIPPrefixMatch(dst[:], ip.String()))
}
