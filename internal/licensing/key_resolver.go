package licensing

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"ad-event-processor/internal/config"
)

func JWTKeyID(tokenStr string) (string, error) {
	parts := strings.Split(tokenStr, ".")
	if len(parts) != 3 {
		return "", ErrInvalidTokenFormat
	}
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return "", ErrInvalidTokenFormat
	}
	var header struct {
		KID string `json:"kid"`
	}
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return "", ErrInvalidTokenFormat
	}
	return strings.TrimSpace(header.KID), nil
}

func ResolvePublicKeyForKID(kid string) (ed25519.PublicKey, error) {
	kid = strings.TrimSpace(kid)
	if kid == "" || kid == DefaultLicenseKeyID {
		return ResolvePublicKey()
	}
	if config.LicensePublicKeyProductionEmbeddedOnly() {
		return nil, errors.New("license public key not found for kid " + kid)
	}
	for _, path := range cohortPublicKeyPaths(kid) {
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
	return nil, errors.New("license public key not found for kid " + kid)
}

func cohortPublicKeyPaths(kid string) []string {
	kid = strings.TrimSpace(kid)
	if kid == "" {
		return nil
	}
	var paths []string
	if root := strings.TrimSpace(os.Getenv("ROOT")); root != "" {
		paths = append(paths, filepath.Join(root, "deploy", "vendor", "keys", kid, "license_public.key"))
	}
	if cwd, err := os.Getwd(); err == nil {
		paths = append(paths, filepath.Join(cwd, "deploy", "vendor", "keys", kid, "license_public.key"))
	}
	paths = append(paths,
		filepath.Join("deploy", "vendor", "keys", kid, "license_public.key"),
		filepath.Join("deploy", "vendor", "license_"+kid+"_public.key"),
	)
	return paths
}

func cohortPrivateKeyPaths(kid string) []string {
	kid = strings.TrimSpace(kid)
	if kid == "" {
		return nil
	}
	var paths []string
	if root := strings.TrimSpace(os.Getenv("ROOT")); root != "" {
		paths = append(paths, filepath.Join(root, "deploy", "vendor", "keys", kid, "license_private.key"))
	}
	if cwd, err := os.Getwd(); err == nil {
		paths = append(paths, filepath.Join(cwd, "deploy", "vendor", "keys", kid, "license_private.key"))
	}
	paths = append(paths,
		filepath.Join("deploy", "vendor", "keys", kid, "license_private.key"),
		filepath.Join("deploy", "vendor", "license_"+kid+"_private.key"),
	)
	return paths
}

func ResolvePrivateKeyFileForKID(kid, explicitPath string) string {
	if p := strings.TrimSpace(explicitPath); p != "" {
		return p
	}
	if env := strings.TrimSpace(config.LicenseEnv("PRIVATE_KEY_FILE")); env != "" {
		return env
	}
	for _, path := range cohortPrivateKeyPaths(kid) {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	if kid == "" || kid == DefaultLicenseKeyID {
		return "deploy/vendor/license_private.key"
	}
	return filepath.Join("deploy", "vendor", "keys", kid, "license_private.key")
}

func VerifyJWTResolved(tokenStr string) (*LicenseClaims, error) {
	kid, err := JWTKeyID(tokenStr)
	if err != nil {
		return nil, err
	}
	pub, err := ResolvePublicKeyForKID(kid)
	if err != nil {
		return nil, err
	}
	return VerifyJWT(tokenStr, pub)
}

func VerifyJWTWithKey(tokenStr string, pubKey ed25519.PublicKey) (*LicenseClaims, error) {
	if len(pubKey) > 0 {
		return VerifyJWT(tokenStr, pubKey)
	}
	return VerifyJWTResolved(tokenStr)
}
