package piihash

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/bidshard/ad-event-processor/internal/config"

	"github.com/minio/highwayhash"
)

const keySize = 32

const (
	fieldIP     byte = 1
	fieldUA     byte = 2
	fieldUserID byte = 3
	fieldSubnet byte = 4
)

type Hasher struct {
	version uint8
	key     [keySize]byte
}

func New(version uint8, key [keySize]byte) *Hasher {
	return &Hasher{version: version, key: key}
}

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

func (h *Hasher) Version() uint8 {
	if h == nil {
		return 0
	}
	return h.version
}

func (h *Hasher) HashIP(ip string) [16]byte {
	return h.hash(fieldIP, ip)
}

func (h *Hasher) HashUA(ua string) [16]byte {
	return h.hash(fieldUA, ua)
}

func (h *Hasher) HashUserID(userID string) [16]byte {
	return h.hash(fieldUserID, userID)
}

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
	sum := sha256.Sum256([]byte("ad_event_processor:pii:salt:v1:" + fallbackSecret))
	copy(key[:], sum[:])
	return key, nil
}

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
