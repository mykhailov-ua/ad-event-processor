// Package piihash provides SIMD-accelerated PII hashing for ClickHouse batch inserts (GAP-DATA-01).
package piihash

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"

	"espx/internal/config"

	"github.com/minio/highwayhash"
)

const keySize = 32

// Field kind bytes distinguish hash domains inside one salt version.
const (
	fieldIP      byte = 1
	fieldUA      byte = 2
	fieldUserID  byte = 3
	fieldSubnet  byte = 4
)

// Hasher hashes PII with a versioned 32-byte salt for ClickHouse storage.
type Hasher struct {
	version uint8
	key     [keySize]byte
}

// New builds a hasher from an explicit salt version and 32-byte key.
func New(version uint8, key [keySize]byte) *Hasher {
	return &Hasher{version: version, key: key}
}

// NewFromConfig loads the active salt from processor config.
func NewFromConfig(cfg *config.Config) (*Hasher, error) {
	if cfg == nil {
		return nil, errors.New("piihash: nil config")
	}
	key, err := decodeSaltKey(string(cfg.PIISaltHex), string(cfg.TokenSymmetricKey))
	if err != nil {
		return nil, err
	}
	version := cfg.PIISaltVersion
	if version == 0 {
		version = 1
	}
	return New(version, key), nil
}

// Version returns the active salt generation.
func (h *Hasher) Version() uint8 {
	if h == nil {
		return 0
	}
	return h.version
}

// HashIP returns a 128-bit HighwayHash of an IP address.
func (h *Hasher) HashIP(ip string) [16]byte {
	return h.hash(fieldIP, ip)
}

// HashUA returns a 128-bit HighwayHash of a user agent string.
func (h *Hasher) HashUA(ua string) [16]byte {
	return h.hash(fieldUA, ua)
}

// HashUserID returns a 128-bit HighwayHash of a user identifier.
func (h *Hasher) HashUserID(userID string) [16]byte {
	return h.hash(fieldUserID, userID)
}

// HashSubnet returns a 128-bit HighwayHash of a masked subnet label.
func (h *Hasher) HashSubnet(subnet string) [16]byte {
	return h.hash(fieldSubnet, subnet)
}

func (h *Hasher) hash(kind byte, value string) [16]byte {
	if h == nil || value == "" {
		return [16]byte{}
	}
	var buf [512]byte
	n := 2
	buf[0] = h.version
	buf[1] = kind
	if len(value) > len(buf)-2 {
		value = value[:len(buf)-2]
	}
	n += copy(buf[2:], value)
	return highwayhash.Sum128(buf[:n], h.key[:])
}

func decodeSaltKey(saltHex, fallbackSecret string) ([keySize]byte, error) {
	var key [keySize]byte
	if saltHex != "" {
		raw, err := hex.DecodeString(saltHex)
		if err != nil {
			return key, fmt.Errorf("piihash: PII_SALT_HEX: %w", err)
		}
		if len(raw) != keySize {
			return key, fmt.Errorf("piihash: PII_SALT_HEX must be %d bytes (%d hex chars)", keySize, keySize*2)
		}
		copy(key[:], raw)
		return key, nil
	}
	if fallbackSecret == "" {
		return key, errors.New("piihash: PII_SALT_HEX or TOKEN_SYMMETRIC_KEY required")
	}
	sum := sha256.Sum256([]byte("espx:pii:salt:v1:" + fallbackSecret))
	copy(key[:], sum[:])
	return key, nil
}

// FixedString16 encodes a 128-bit hash for ClickHouse FixedString(16) columns.
func FixedString16(h [16]byte) string {
	return string(h[:])
}
func TestHasher() *Hasher {
	var key [keySize]byte
	for i := range key {
		key[i] = byte(i + 1)
	}
	return New(1, key)
}
