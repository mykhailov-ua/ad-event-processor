package domains

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"ad-event-processor/internal/config"
)

func ingressTLSDir(cfg *config.Config) string {
	if cfg != nil {
		dir := strings.TrimSpace(cfg.Management.IngressTLSDir)
		if dir != "" {
			return dir
		}
	}
	return "deploy/ingress/certs"
}

func wildcardIngressCertPaths(cfg *config.Config, zoneName string) (certPath, keyPath, accountKeyPath string) {
	dir := ingressTLSDir(cfg)
	safeZone := strings.ReplaceAll(normalizeZoneName(zoneName), "/", "_")
	base := filepath.Join(dir, "wildcard-"+safeZone)
	return base + ".crt", base + ".key", base + ".acme-account.key"
}

func deployWildcardIngressTLS(cfg *config.Config, zoneName string, certPEM, keyPEM []byte) error {
	if len(certPEM) == 0 || len(keyPEM) == 0 {
		return fmt.Errorf("wildcard ingress deploy: empty certificate material")
	}
	certPath, keyPath, _ := wildcardIngressCertPaths(cfg, zoneName)
	dir := filepath.Dir(certPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("wildcard ingress deploy: mkdir %s: %w", dir, err)
	}
	if err := os.WriteFile(certPath, certPEM, 0o644); err != nil {
		return fmt.Errorf("wildcard ingress deploy: write cert: %w", err)
	}
	if err := os.WriteFile(keyPath, keyPEM, 0o600); err != nil {
		return fmt.Errorf("wildcard ingress deploy: write key: %w", err)
	}
	return nil
}
