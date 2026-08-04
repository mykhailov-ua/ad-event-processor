package licensing

import (
	"crypto/ed25519"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"espx/internal/config"
)

const defaultPublicKeyRelPath = "deploy/vendor/license_public.key"

// Production verify key shipped in release binaries (not secret).
const embeddedProductionPublicKeyHex = "ede21d8e759af2ba68a74149d28f37a859d33497accee01e8f8ac712bd455c70"

// ResolvePublicKey loads the Ed25519 verify key from ESPX_LICENSE_PUBLIC_KEY or file paths.
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
	if pub, err := ParsePublicKey([]byte(embeddedProductionPublicKeyHex)); err == nil {
		return pub, nil
	}
	return nil, errors.New("license public key not configured (set ESPX_LICENSE_PUBLIC_KEY or deploy/vendor/license_public.key)")
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
