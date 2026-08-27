package licensing

import (
	"crypto/ed25519"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/licensing/embedkey"
)

const defaultPublicKeyRelPath = "deploy/vendor/license_public.key"

var (
	embeddedPubOnce    sync.Once
	embeddedPub        ed25519.PublicKey
	pubkeyOverrideWarn sync.Once
)

func embeddedProductionPublicKey() ed25519.PublicKey {
	embeddedPubOnce.Do(func() {
		embeddedPub = embedkey.EmbeddedProductionPublicKey()
	})
	return embeddedPub
}

func ResolvePublicKey() (ed25519.PublicKey, error) {
	if config.LicensePublicKeyProductionEmbeddedOnly() {
		return embeddedProductionPublicKeyOrError()
	}
	if config.LicensePublicKeyOverrideAllowed() {
		pubkeyOverrideWarn.Do(func() {
			slog.Warn("license public key override enabled; do not use in production appliances")
		})
	}
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
	return embeddedProductionPublicKeyOrError()
}

func embeddedProductionPublicKeyOrError() (ed25519.PublicKey, error) {
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
