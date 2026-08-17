package ingestion

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"net"
	"strings"
	"time"

	"github.com/bidshard/ad-event-processor/internal/domain"
	"github.com/google/uuid"
)

const (
	attestationTokenVersion   = 1
	attestationPayloadLen     = 49
	attestationMACLen         = 16
	attestationTokenBinaryLen = attestationPayloadLen + attestationMACLen
	attestationTokenB64Len    = 87 // base64.RawURLEncoding.EncodedLen(attestationTokenBinaryLen)
	attestationCookiePrefix   = "Attestation-Token="
	attestationDefaultTTL     = 300
	attestationMinTTL         = 60
	attestationMaxTTL         = 900
)

type attestationHMACKey struct {
	secret []byte
	ipad   [linkHMACBlockSize]byte
	opad   [linkHMACBlockSize]byte
}

// MintAttestationToken builds a base64url cookie value (cold path).
func MintAttestationToken(secret []byte, campaignID uuid.UUID, clientIP string, ttlSec int32, nowUnix int64) (string, error) {
	if len(secret) == 0 {
		return "", errAttestationNoSecret
	}
	ttl := clampAttestationTTL(ttlSec)
	expires := nowUnix + int64(ttl)
	var payload [attestationPayloadLen]byte
	payload[0] = attestationTokenVersion
	copy(payload[1:17], campaignID[:])
	putInt64BE(payload[17:25], nowUnix)
	putInt64BE(payload[25:33], expires)
	if !encodeAttestationIPPrefix(clientIP, payload[33:49]) {
		return "", errAttestationBadIP
	}
	mac := attestationMACStatic(secret, payload[:])
	var raw [attestationTokenBinaryLen]byte
	copy(raw[:attestationPayloadLen], payload[:])
	copy(raw[attestationPayloadLen:], mac)
	return base64.RawURLEncoding.EncodeToString(raw[:]), nil
}

func (h *AdsPacketHandler) verifyAttestationCookie(cookieHeader []byte, campaignID uuid.UUID, clientIP string, nowUnix int64) bool {
	if h == nil || len(h.attestationKeys) == 0 {
		return false
	}
	token := extractAttestationCookie(cookieHeader)
	if len(token) == 0 {
		return false
	}
	var decoded [attestationTokenBinaryLen]byte
	n, ok := decodeAttestationTokenBase64URL(token, decoded[:])
	if !ok || n != attestationTokenBinaryLen {
		return false
	}
	payload := decoded[:attestationPayloadLen]
	gotMAC := decoded[attestationPayloadLen:]
	if payload[0] != attestationTokenVersion {
		return false
	}
	var gotCampaign uuid.UUID
	copy(gotCampaign[:], payload[1:17])
	if gotCampaign != campaignID {
		return false
	}
	expires := int64BE(payload[25:33])
	if expires <= nowUnix {
		return false
	}
	if !attestationIPPrefixMatch(payload[33:49], clientIP) {
		return false
	}
	for i := range h.attestationKeys {
		var expect [attestationMACLen]byte
		if !attestationMACIntoPads(&h.attestationKeys[i].ipad, &h.attestationKeys[i].opad, h.attestationInnerScratch[:], payload, &expect) {
			continue
		}
		if subtle.ConstantTimeCompare(expect[:], gotMAC) == 1 {
			return true
		}
	}
	return false
}

func (h *AdsPacketHandler) ConfigureAttestation(secrets [][]byte) {
	if h == nil {
		return
	}
	h.attestationKeys = h.attestationKeys[:0]
	for _, secret := range secrets {
		if len(secret) == 0 {
			continue
		}
		var key attestationHMACKey
		key.secret = append([]byte(nil), secret...)
		linkInitHMACPads(key.secret, &key.ipad, &key.opad)
		h.attestationKeys = append(h.attestationKeys, key)
	}
}

func (h *AdsPacketHandler) attestationRequired(campaignID uuid.UUID) bool {
	if h == nil || h.registry == nil || len(h.attestationKeys) == 0 {
		return false
	}
	camp, ok := h.registry.GetCampaign(campaignID)
	return ok && camp != nil && camp.AttestationEnabled && camp.SafePageEnabled
}

func buildAttestationSetCookie(token string, maxAge int32) []byte {
	if token == "" || maxAge <= 0 {
		return nil
	}
	// Set-Cookie: Attestation-Token=...; Path=/click; Max-Age=N; HttpOnly; SameSite=Lax
	prefix := []byte("Set-Cookie: Attestation-Token=")
	suffix := []byte("; Path=/click; Max-Age=")
	mid := []byte(token)
	tail := []byte("; HttpOnly; SameSite=Lax\r\n")
	var ageScratch [12]byte
	age := appendInt64(ageScratch[:0], int64(maxAge))
	out := make([]byte, 0, len(prefix)+len(mid)+len(suffix)+len(age)+len(tail))
	out = append(out, prefix...)
	out = append(out, mid...)
	out = append(out, suffix...)
	out = append(out, age...)
	out = append(out, tail...)
	return out
}

func (h *AdsPacketHandler) mintAttestationCookie(campaignID uuid.UUID, clientIP string) (string, int32) {
	if !h.attestationRequired(campaignID) {
		return "", 0
	}
	camp, _ := h.registry.GetCampaign(campaignID)
	ttl := campaignAttestationTTL(camp)
	token, err := MintAttestationToken(h.attestationKeys[0].secret, campaignID, clientIP, ttl, time.Now().Unix())
	if err != nil {
		return "", 0
	}
	return token, ttl
}

func campaignAttestationTTL(camp *domain.Campaign) int32 {
	if camp == nil || camp.AttestationTTLSec <= 0 {
		return attestationDefaultTTL
	}
	return clampAttestationTTL(camp.AttestationTTLSec)
}

func clampAttestationTTL(ttl int32) int32 {
	if ttl < attestationMinTTL {
		return attestationMinTTL
	}
	if ttl > attestationMaxTTL {
		return attestationMaxTTL
	}
	return ttl
}

func extractAttestationCookie(cookieHeader []byte) []byte {
	if len(cookieHeader) == 0 {
		return nil
	}
	prefix := []byte(attestationCookiePrefix)
	start := 0
	for start < len(cookieHeader) {
		idx := bytesIndex(cookieHeader[start:], prefix)
		if idx < 0 {
			return nil
		}
		pos := start + idx + len(prefix)
		end := pos
		for end < len(cookieHeader) && cookieHeader[end] != ';' {
			end++
		}
		val := trimHTTPVal(cookieHeader[pos:end])
		if len(val) > 0 {
			return val
		}
		start = end + 1
	}
	return nil
}

func bytesIndex(b, sub []byte) int {
	if len(sub) == 0 {
		return 0
	}
	if len(b) < len(sub) {
		return -1
	}
	for i := 0; i+len(sub) <= len(b); i++ {
		if bytesEqual(b[i:i+len(sub)], unsafeString(sub)) {
			return i
		}
	}
	return -1
}

func attestationMACStatic(secret, payload []byte) []byte {
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write(payload)
	var scratch [sha256.Size]byte
	return mac.Sum(scratch[:0])[:attestationMACLen]
}

func attestationMACIntoPads(ipad, opad *[linkHMACBlockSize]byte, scratch []byte, payload []byte, out *[attestationMACLen]byte) bool {
	need := linkHMACBlockSize + len(payload)
	if need > len(scratch) || out == nil {
		return false
	}
	copy(scratch[:linkHMACBlockSize], ipad[:])
	copy(scratch[linkHMACBlockSize:], payload)
	inner := sha256.Sum256(scratch[:need])
	var outerBuf [linkHMACBlockSize + sha256.Size]byte
	copy(outerBuf[:linkHMACBlockSize], opad[:])
	copy(outerBuf[linkHMACBlockSize:], inner[:])
	outer := sha256.Sum256(outerBuf[:])
	copy(out[:], outer[:attestationMACLen])
	return true
}

func encodeAttestationIPPrefix(ip string, dst []byte) bool {
	if len(dst) < 16 {
		return false
	}
	parsed := net.ParseIP(strings.TrimSpace(ip))
	if parsed == nil {
		return false
	}
	if v4 := parsed.To4(); v4 != nil {
		for i := 0; i < 10; i++ {
			dst[i] = 0
		}
		dst[10] = 0xff
		dst[11] = 0xff
		copy(dst[12:], v4)
		return true
	}
	v6 := parsed.To16()
	copy(dst[:8], v6[:8])
	for i := 8; i < 16; i++ {
		dst[i] = 0
	}
	return true
}

func attestationIPPrefixMatch(stored []byte, ip string) bool {
	if len(stored) < 16 {
		return false
	}
	parsed := net.ParseIP(strings.TrimSpace(ip))
	if parsed == nil {
		return false
	}
	if v4 := parsed.To4(); v4 != nil {
		var want [16]byte
		if !encodeAttestationIPPrefix(v4.String(), want[:]) {
			return false
		}
		return subtle.ConstantTimeCompare(stored[:16], want[:]) == 1
	}
	v6 := parsed.To16()
	return subtle.ConstantTimeCompare(stored[:8], v6[:8]) == 1
}

func decodeAttestationTokenBase64URL(src []byte, dst []byte) (int, bool) {
	if len(src) != attestationTokenB64Len || len(dst) < attestationTokenBinaryLen {
		return 0, false
	}
	var scratch [attestationTokenBinaryLen]byte
	n, err := base64.RawURLEncoding.Decode(scratch[:], src)
	if err != nil || n != attestationTokenBinaryLen {
		return 0, false
	}
	copy(dst, scratch[:n])
	return n, true
}

func putInt64BE(dst []byte, v int64) {
	for i := 7; i >= 0; i-- {
		dst[i] = byte(v)
		v >>= 8
	}
}

func int64BE(src []byte) int64 {
	var v int64
	for i := 0; i < 8 && i < len(src); i++ {
		v = (v << 8) | int64(src[i])
	}
	return v
}

var (
	errAttestationNoSecret = errAttestation("missing attestation secret")
	errAttestationBadIP    = errAttestation("invalid client ip")
)

type errAttestation string

func (e errAttestation) Error() string { return string(e) }
