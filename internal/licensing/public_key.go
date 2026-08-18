package licensing

import (
	"crypto/ed25519"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/bidshard/ad-event-processor/internal/config"
)

const defaultPublicKeyRelPath = "deploy/vendor/license_public.key"

var (
	embeddedPubKeyMasked = [ed25519.PublicKeySize]byte{
		0x4a, 0xde, 0x8c, 0xd0, 0x57, 0x61, 0xe6, 0x32,
		0x05, 0xa6, 0x8f, 0x0e, 0x6b, 0xfd, 0x07, 0x75,
		0x16, 0x59, 0x57, 0x8e, 0x49, 0x98, 0x9b, 0xda,
		0x8d, 0x27, 0xff, 0x63, 0x23, 0x60, 0x0c, 0xf6,
	}
	embeddedPubKeyMask = [ed25519.PublicKeySize]byte{
		0xa7, 0x3c, 0x91, 0x5e, 0x22, 0xfb, 0x14, 0x88,
		0x6d, 0x01, 0xce, 0x47, 0xb9, 0x72, 0x30, 0xdd,
		0x4f, 0x8a, 0x63, 0x19, 0xe5, 0x56, 0x7b, 0xc4,
		0x02, 0xad, 0x38, 0x71, 0x9e, 0x25, 0x50, 0x86,
	}
	embeddedPubOnce sync.Once
	embeddedPub     ed25519.PublicKey
)

func embeddedProductionPublicKey() ed25519.PublicKey {
	embeddedPubOnce.Do(func() {
		var raw [ed25519.PublicKeySize]byte
		for i := range raw {
			raw[i] = embeddedPubKeyMasked[i] ^ embeddedPubKeyMask[i]
		}
		embeddedPub = ed25519.PublicKey(raw[:])
	})
	return embeddedPub
}

func ResolvePublicKey() (ed25519.PublicKey, error) {
	if raw := strings.TrimSpace(config.LicenseEnv("PUBLIC_KEY")); raw != "" {
		return ParsePublicKey([]byte(raw))
	}
	for _, path := range publicKeySearchPaths() {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		pub, err := ParsePublicKey(data)
		if err != nil {
			continue
		}
		return pub, nil
	}
	if pub := embeddedProductionPublicKey(); len(pub) == ed25519.PublicKeySize {
		return pub, nil
	}
	return nil, errors.New("license public key not configured (set AD_EVENT_PROCESSOR_LICENSE_PUBLIC_KEY or deploy/vendor/license_public.key)")
}

func publicKeySearchPaths() []string {
	var paths []string
	if file := strings.TrimSpace(config.LicenseEnv("PUBLIC_KEY_FILE")); file != "" {
		paths = append(paths, file)
	}
	if root := strings.TrimSpace(os.Getenv("ROOT")); root != "" {
		paths = append(paths, filepath.Join(root, defaultPublicKeyRelPath))
	}
	if cwd, err := os.Getwd(); err == nil {
		paths = append(paths, filepath.Join(cwd, defaultPublicKeyRelPath))
	}
	paths = append(paths, defaultPublicKeyRelPath)
	return paths
}
