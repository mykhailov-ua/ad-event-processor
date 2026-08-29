package track

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"time"

	"ad-event-processor/internal/domain"
)

const (
	LinkSigHexLen                = 32
	LinkSigMACBytes              = 16
	LinkSigningMaxTTL            = 3600
	LinkSigningTTLAttestationCap = 300
	LinkHMACBlockSize            = 64
	LinkSignInnerScratchLen      = LinkHMACBlockSize + maxClickQueryValue + 1 + 20
)

var (
	linkSignColon  = [...]byte{':'}
	linkSignDigit0 = [...]byte{'0'}
)

func AppendLinkSignature(dst, secret []byte, clickID []byte, expiresUnix int64) []byte {
	if len(secret) == 0 || len(clickID) == 0 {
		return dst
	}
	sep := byte('&')
	if len(dst) == 0 {
		return dst
	}
	if dst[len(dst)-1] != '?' && !containsQuery(dst) {
		sep = '?'
	}
	dst = append(dst, sep)
	dst = append(dst, 'e', 'x', 'p', 'i', 'r', 'e', 's', '=')
	dst = appendInt64(dst, expiresUnix)
	dst = append(dst, '&', '_', 's', 'i', 'g', '=')
	sig := LinkSignMACStatic(secret, clickID, expiresUnix)
	dst = linkAppendHex16(dst, sig)
	return dst
}

func containsQuery(b []byte) bool {
	for i := range len(b) {
		if b[i] == '?' {
			return true
		}
	}
	return false
}

func LinkSignMACStatic(secret, clickID []byte, expiresUnix int64) []byte {
	mac := hmac.New(sha256.New, secret)
	mac.Write(clickID)
	mac.Write(linkSignColon[:])
	writeInt64ToMAC(mac, expiresUnix)
	var scratch [sha256.Size]byte
	return mac.Sum(scratch[:0])[:LinkSigMACBytes]
}

func LinkInitHMACPads(secret []byte, ipad, opad *[LinkHMACBlockSize]byte) {
	var keyBlock [LinkHMACBlockSize]byte
	key := secret
	if len(key) > LinkHMACBlockSize {
		sum := sha256.Sum256(secret)
		key = sum[:]
	}
	copy(keyBlock[:], key)
	for i := range LinkHMACBlockSize {
		ipad[i] = keyBlock[i] ^ 0x36
		opad[i] = keyBlock[i] ^ 0x5c
	}
}

func linkWriteInt64Scratch(buf []byte, off int, v int64) int {
	if v == 0 {
		buf[off] = '0'
		return off + 1
	}
	var digitBuf [20]byte
	i := len(digitBuf)
	for v > 0 {
		i--
		digitBuf[i] = byte('0' + v%10)
		v /= 10
	}
	n := copy(buf[off:], digitBuf[i:])
	return off + n
}

func LinkSignMACIntoPads(ipad, opad *[LinkHMACBlockSize]byte, scratch []byte, clickID []byte, expiresUnix int64, out *[LinkSigMACBytes]byte) bool {
	need := LinkHMACBlockSize + len(clickID) + 1 + 20
	if need > len(scratch) || out == nil {
		return false
	}
	copy(scratch[:LinkHMACBlockSize], ipad[:])
	off := LinkHMACBlockSize
	off += copy(scratch[off:], clickID)
	scratch[off] = ':'
	off++
	off = linkWriteInt64Scratch(scratch, off, expiresUnix)
	inner := sha256.Sum256(scratch[:off])
	var outerBuf [LinkHMACBlockSize + sha256.Size]byte
	copy(outerBuf[:LinkHMACBlockSize], opad[:])
	copy(outerBuf[LinkHMACBlockSize:], inner[:])
	outer := sha256.Sum256(outerBuf[:])
	copy(out[:], outer[:LinkSigMACBytes])
	return true
}

func writeInt64ToMAC(mac interface {
	Write(p []byte) (n int, err error)
}, v int64,
) {
	if v == 0 {
		_, _ = mac.Write(linkSignDigit0[:])
		return
	}
	var buf [20]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	_, _ = mac.Write(buf[i:])
}

func appendInt64(dst []byte, v int64) []byte {
	if v == 0 {
		return append(dst, '0')
	}
	if v < 0 {
		dst = append(dst, '-')
		v = -v
	}
	var buf [20]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	return append(dst, buf[i:]...)
}

func linkAppendHex16(dst, src []byte) []byte {
	const hexdigits = "0123456789abcdef"
	n := len(src)
	if n == 0 {
		return dst
	}
	_ = src[n-1]
	for i := range n {
		b := src[i]
		dst = append(dst, hexdigits[b>>4], hexdigits[b&0x0f])
	}
	return dst
}

func VerifyLinkSignature(secret, clickID, sig []byte, expiresUnix, nowUnix int64) bool {
	if len(secret) == 0 || len(clickID) == 0 || expiresUnix <= 0 {
		return false
	}
	if nowUnix > expiresUnix {
		return false
	}
	if expiresUnix-nowUnix > LinkSigningMaxTTL {
		return false
	}
	if len(sig) != LinkSigHexLen {
		return false
	}
	expected := LinkSignMACStatic(secret, clickID, expiresUnix)
	var got [LinkSigMACBytes]byte
	if !DecodeHex32Into(sig, &got) {
		return false
	}
	return subtle.ConstantTimeCompare(got[:], expected) == 1
}

func DecodeHex32Into(b []byte, out *[LinkSigMACBytes]byte) bool {
	if len(b) != LinkSigHexLen {
		return false
	}
	for i := range LinkSigMACBytes {
		hi, ok1 := linkHexByte(b[2*i])
		lo, ok2 := linkHexByte(b[2*i+1])
		if !ok1 || !ok2 {
			return false
		}
		out[i] = hi<<4 | lo
	}
	return true
}

func linkHexByte(c byte) (byte, bool) {
	switch {
	case c >= '0' && c <= '9':
		return c - '0', true
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10, true
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10, true
	default:
		return 0, false
	}
}

func LinkSigningExpires(now time.Time, ttlSec int32) int64 {
	if ttlSec <= 0 {
		ttlSec = 900
	}
	if ttlSec > int32(LinkSigningMaxTTL) {
		ttlSec = int32(LinkSigningMaxTTL)
	}
	return now.Unix() + int64(ttlSec)
}

func EffectiveLinkSigningTTLSec(camp *domain.Campaign) int32 {
	ttl := int32(900)
	if camp != nil && camp.LinkSigningTTLSec > 0 {
		ttl = camp.LinkSigningTTLSec
	}
	if camp != nil && domain.ResolveAttestationMode(camp.AttestationMode, camp.AttestationEnabled).RequiresProbe() {
		if ttl > LinkSigningTTLAttestationCap {
			ttl = LinkSigningTTLAttestationCap
		}
	}
	return ttl
}

func ParseLinkExpires(b []byte) (int64, bool) {
	if len(b) == 0 {
		return 0, false
	}
	var n int64
	for i := range len(b) {
		c := b[i]
		if c < '0' || c > '9' {
			return 0, false
		}
		n = n*10 + int64(c-'0')
	}
	return n, true
}
