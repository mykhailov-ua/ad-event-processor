package parser

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"net"
	"strings"

	"github.com/google/uuid"
)

const HMACBlockSize = 64

const (
	attestationTokenVersion   = 1
	attestationPayloadLen     = 49
	attestationMACLen         = 16
	attestationTokenBinaryLen = attestationPayloadLen + attestationMACLen
	attestationTokenB64Len    = 87
	AttestationCookiePrefix   = "Attestation-Token="
	attestationDefaultTTL     = 300
	attestationMinTTL         = 60
	attestationMaxTTL         = 900
)

type AttestationHMACKey struct {
	Secret []byte
	Ipad   [HMACBlockSize]byte
	Opad   [HMACBlockSize]byte
}

func MintAttestationToken(secret []byte, campaignID uuid.UUID, clientIP string, ttlSec int32, nowUnix int64) (string, error) {
	if len(secret) == 0 {
		return "", errAttestationNoSecret
	}
	ttl := ClampAttestationTTL(ttlSec)
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

func ExtractAttestationCookie(cookieHeader []byte) []byte {
	if len(cookieHeader) == 0 {
		return nil
	}
	prefix := []byte(AttestationCookiePrefix)
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
		val := cookieHeader[pos:end]
		if len(val) > 0 {
			return val
		}
		start = end + 1
	}
	return nil
}

func VerifyAttestationToken(keys []AttestationHMACKey, scratch []byte, cookieHeader []byte, campaignID uuid.UUID, clientIP string, nowUnix int64) bool {
	if len(keys) == 0 {
		return false
	}
	token := ExtractAttestationCookie(cookieHeader)
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
	for i := range keys {
		var expect [attestationMACLen]byte
		if !AttestationMACIntoPads(&keys[i].Ipad, &keys[i].Opad, scratch, payload, &expect) {
			continue
		}
		if subtle.ConstantTimeCompare(expect[:], gotMAC) == 1 {
			return true
		}
	}
	return false
}

func ClampAttestationTTL(ttl int32) int32 {
	if ttl < attestationMinTTL {
		return attestationMinTTL
	}
	if ttl > attestationMaxTTL {
		return attestationMaxTTL
	}
	return ttl
}

func AttestationMACIntoPads(ipad, opad *[HMACBlockSize]byte, scratch []byte, payload []byte, out *[attestationMACLen]byte) bool {
	need := HMACBlockSize + len(payload)
	if need > len(scratch) || out == nil {
		return false
	}
	copy(scratch[:HMACBlockSize], ipad[:])
	copy(scratch[HMACBlockSize:], payload)
	inner := sha256.Sum256(scratch[:need])
	var outerBuf [HMACBlockSize + sha256.Size]byte
	copy(outerBuf[:HMACBlockSize], opad[:])
	copy(outerBuf[HMACBlockSize:], inner[:])
	outer := sha256.Sum256(outerBuf[:])
	copy(out[:], outer[:attestationMACLen])
	return true
}

func bytesIndex(b, sub []byte) int {
	if len(sub) == 0 {
		return 0
	}
	if len(b) < len(sub) {
		return -1
	}
	for i := 0; i+len(sub) <= len(b); i++ {
		match := true
		for j := range sub {
			if b[i+j] != sub[j] {
				match = false
				break
			}
		}
		if match {
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

func encodeAttestationIPPrefix(ip string, dst []byte) bool {
	if len(dst) < 16 {
		return false
	}
	parsed := net.ParseIP(strings.TrimSpace(ip))
	if parsed == nil {
		return false
	}
	if v4 := parsed.To4(); v4 != nil {
		for i := range 10 {
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
	errAttestationNoSecret = errors.New("missing attestation secret")
	errAttestationBadIP    = errors.New("invalid client ip")
)
