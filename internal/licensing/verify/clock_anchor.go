package verify

import (
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/binary"
	"errors"
	"os"
	"time"
)

var (
	ErrClockAnchorRewind = errors.New("license clock anchor rewind")
	ErrClockAnchorTamper = errors.New("license clock anchor tamper")
)

const (
	clockAnchorMagic   = "aedclk01"
	clockAnchorBodyLen = 8 + 8 + 8
	clockAnchorMACLen  = sha256.Size
	clockAnchorFileLen = len(clockAnchorMagic) + clockAnchorBodyLen + clockAnchorMACLen
)

func ClockAnchorPath(licensePath string) string {
	return licensePath + ".clock"
}

func CheckClockAnchor(licensePath string, pubKey ed25519.PublicKey, hostFingerprint string, now time.Time, threshold time.Duration) error {
	if threshold <= 0 {
		threshold = 5 * time.Minute
	}
	anchorPath := ClockAnchorPath(licensePath)
	raw, err := os.ReadFile(anchorPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	wallUnix, err := decodeClockAnchor(licensePath, pubKey, hostFingerprint, raw)
	if err != nil {
		return err
	}
	nowUnix := now.UTC().Unix()
	if nowUnix+int64(threshold.Seconds()) < wallUnix {
		return ErrClockAnchorRewind
	}
	return nil
}

func UpdateClockAnchor(licensePath string, pubKey ed25519.PublicKey, hostFingerprint string, now time.Time, mono time.Duration) error {
	mckWork, err := DeriveMCKWorkForRecheckFromLicenseFile(licensePath, pubKey, hostFingerprint)
	if err != nil {
		return err
	}
	wallUnix := now.UTC().Unix()
	anchorPath := ClockAnchorPath(licensePath)
	if raw, readErr := os.ReadFile(anchorPath); readErr == nil {
		if stored, decErr := decodeClockAnchorWithKey(mckWork, raw); decErr == nil && stored > wallUnix {
			wallUnix = stored
		}
	}
	if mono < 0 {
		mono = 0
	}
	payload := encodeClockAnchorBody(wallUnix, mono)
	mac := hmac.New(sha256.New, mckWork[:])
	_, _ = mac.Write([]byte(clockAnchorMagic))
	_, _ = mac.Write(payload)
	sum := mac.Sum(nil)
	out := make([]byte, 0, clockAnchorFileLen)
	out = append(out, clockAnchorMagic...)
	out = append(out, payload...)
	out = append(out, sum...)
	return WriteFileAtomic(anchorPath, out, 0o600)
}

func decodeClockAnchor(licensePath string, pubKey ed25519.PublicKey, hostFingerprint string, raw []byte) (int64, error) {
	mckWork, err := DeriveMCKWorkForRecheckFromLicenseFile(licensePath, pubKey, hostFingerprint)
	if err != nil {
		return 0, err
	}
	return decodeClockAnchorWithKey(mckWork, raw)
}

func decodeClockAnchorWithKey(mckWork [32]byte, raw []byte) (int64, error) {
	if len(raw) != clockAnchorFileLen {
		return 0, ErrClockAnchorTamper
	}
	if string(raw[:len(clockAnchorMagic)]) != clockAnchorMagic {
		return 0, ErrClockAnchorTamper
	}
	body := raw[len(clockAnchorMagic) : len(clockAnchorMagic)+clockAnchorBodyLen]
	storedMAC := raw[len(clockAnchorMagic)+clockAnchorBodyLen:]
	mac := hmac.New(sha256.New, mckWork[:])
	_, _ = mac.Write([]byte(clockAnchorMagic))
	_, _ = mac.Write(body)
	if subtle.ConstantTimeCompare(mac.Sum(nil), storedMAC) != 1 {
		return 0, ErrClockAnchorTamper
	}
	return int64(binary.BigEndian.Uint64(body[:8])), nil
}

func encodeClockAnchorBody(wallUnix int64, mono time.Duration) []byte {
	body := make([]byte, clockAnchorBodyLen)
	binary.BigEndian.PutUint64(body[0:8], uint64(wallUnix))
	binary.BigEndian.PutUint64(body[8:16], uint64(mono))
	return body
}
