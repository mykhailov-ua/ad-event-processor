package licensing

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"strings"
)

var (
	buildAssetSealSaltHex   string
	assetSealSaltOverride   []byte
	assetSealSaltOverrideOn bool
)

var ErrInvalidAssetSealSalt = errors.New("invalid asset seal salt")

func DeriveReleaseAssetSealSalt(vaultSalt, tag string) ([32]byte, error) {
	var zero [32]byte
	vaultSalt = strings.TrimSpace(vaultSalt)
	tag = strings.TrimSpace(tag)
	if vaultSalt == "" || tag == "" {
		return zero, ErrInvalidAssetSealSalt
	}
	sum := sha256.Sum256([]byte(vaultSalt + ":" + tag))
	return sum, nil
}

func decodeSealSaltHex(raw string) ([]byte, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, ErrInvalidAssetSealSalt
	}
	b, err := hex.DecodeString(raw)
	if err != nil {
		return nil, ErrInvalidAssetSealSalt
	}
	if len(b) != 32 {
		return nil, ErrInvalidAssetSealSalt
	}
	return b, nil
}

func resolveAssetSealSalt() []byte {
	if assetSealSaltOverrideOn {
		if len(assetSealSaltOverride) == 0 {
			return nil
		}
		out := make([]byte, len(assetSealSaltOverride))
		copy(out, assetSealSaltOverride)
		return out
	}
	if v := strings.TrimSpace(os.Getenv("AD_EVENT_PROCESSOR_ASSET_SEAL_SALT")); v != "" {
		if b, err := decodeSealSaltHex(v); err == nil {
			return b
		}
	}
	if buildAssetSealSaltHex != "" {
		if b, err := decodeSealSaltHex(buildAssetSealSaltHex); err == nil {
			return b
		}
	}
	return nil
}

func SetAssetSealSaltForTest(salt []byte) func() {
	prevOn := assetSealSaltOverrideOn
	prev := assetSealSaltOverride
	assetSealSaltOverrideOn = true
	if len(salt) == 0 {
		assetSealSaltOverride = nil
	} else {
		assetSealSaltOverride = append([]byte(nil), salt...)
	}
	return func() {
		assetSealSaltOverrideOn = prevOn
		assetSealSaltOverride = prev
	}
}
