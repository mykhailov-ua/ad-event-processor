package ingestion

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"time"
)

const (
	linkSigHexLen           = 32
	linkSigMACBytes         = 16
	linkSigningMaxTTL       = 3600
	linkHMACBlockSize       = 64
	linkSignInnerScratchLen = linkHMACBlockSize + maxClickQueryValue + 1 + 20
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
	sig := linkSignMACStatic(secret, clickID, expiresUnix)
	dst = linkAppendHex16(dst, sig)
	return dst
}

func containsQuery(b []byte) bool {
	for i := 0; i < len(b); i++ {
		if b[i] == '?' {
			return true
		}
	}
	return false
}

func linkSignMACStatic(secret, clickID []byte, expiresUnix int64) []byte {
	mac := hmac.New(sha256.New, secret)
	mac.Write(clickID)
	mac.Write(linkSignColon[:])
	writeInt64ToMAC(mac, expiresUnix)
	var scratch [sha256.Size]byte
	return mac.Sum(scratch[:0])[:linkSigMACBytes]
}

func linkInitHMACPads(secret []byte, ipad, opad *[linkHMACBlockSize]byte) {
	var keyBlock [linkHMACBlockSize]byte
	key := secret
	if len(key) > linkHMACBlockSize {
		sum := sha256.Sum256(secret)
		key = sum[:]
	}
	copy(keyBlock[:], key)
	for i := 0; i < linkHMACBlockSize; i++ {
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

func linkSignMACIntoPads(ipad, opad *[linkHMACBlockSize]byte, scratch []byte, clickID []byte, expiresUnix int64, out *[linkSigMACBytes]byte) bool {
	need := linkHMACBlockSize + len(clickID) + 1 + 20
	if need > len(scratch) || out == nil {
		return false
	}
	copy(scratch[:linkHMACBlockSize], ipad[:])
	off := linkHMACBlockSize
	off += copy(scratch[off:], clickID)
	scratch[off] = ':'
	off++
	off = linkWriteInt64Scratch(scratch, off, expiresUnix)
	inner := sha256.Sum256(scratch[:off])
	var outerBuf [linkHMACBlockSize + sha256.Size]byte
	copy(outerBuf[:linkHMACBlockSize], opad[:])
	copy(outerBuf[linkHMACBlockSize:], inner[:])
	outer := sha256.Sum256(outerBuf[:])
	copy(out[:], outer[:linkSigMACBytes])
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

func (h *AdsPacketHandler) linkSignMACInto(clickID []byte, expiresUnix int64, out *[linkSigMACBytes]byte) bool {
	if h == nil || len(h.linkSigningSecret) == 0 || len(clickID) == 0 || out == nil {
		return false
	}
	if linkSignMACIntoPads(&h.linkHMACIpad, &h.linkHMACOpad, h.linkSignInnerScratch[:], clickID, expiresUnix, out) {
		return true
	}
	sig := linkSignMACStatic(h.linkSigningSecret, clickID, expiresUnix)
	copy(out[:], sig)
	return true
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
	for i := 0; i < n; i++ {
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
	if expiresUnix-nowUnix > linkSigningMaxTTL {
		return false
	}
	if len(sig) != linkSigHexLen {
		return false
	}
	expected := linkSignMACStatic(secret, clickID, expiresUnix)
	var got [linkSigMACBytes]byte
	if !decodeHex32Into(sig, &got) {
		return false
	}
	return subtle.ConstantTimeCompare(got[:], expected) == 1
}

func (h *AdsPacketHandler) verifyLinkSignature(clickID, sig []byte, expiresUnix, nowUnix int64) bool {
	if h == nil || len(h.linkSigningSecret) == 0 || len(clickID) == 0 || expiresUnix <= 0 {
		return false
	}
	if nowUnix > expiresUnix {
		return false
	}
	if expiresUnix-nowUnix > linkSigningMaxTTL {
		return false
	}
	if len(sig) != linkSigHexLen {
		return false
	}
	var expected [linkSigMACBytes]byte
	if !h.linkSignMACInto(clickID, expiresUnix, &expected) {
		return false
	}
	var got [linkSigMACBytes]byte
	if !decodeHex32Into(sig, &got) {
		return false
	}
	return subtle.ConstantTimeCompare(got[:], expected[:]) == 1
}

func decodeHex32Into(b []byte, out *[linkSigMACBytes]byte) bool {
	if len(b) != linkSigHexLen {
		return false
	}
	for i := 0; i < linkSigMACBytes; i++ {
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
	if ttlSec > int32(linkSigningMaxTTL) {
		ttlSec = int32(linkSigningMaxTTL)
	}
	return now.Unix() + int64(ttlSec)
}

func parseLinkExpires(b []byte) (int64, bool) {
	if len(b) == 0 {
		return 0, false
	}
	var n int64
	for i := 0; i < len(b); i++ {
		c := b[i]
		if c < '0' || c > '9' {
			return 0, false
		}
		n = n*10 + int64(c-'0')
	}
	return n, true
}
