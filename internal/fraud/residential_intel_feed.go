package fraud

import (
	"fmt"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
)

const externalResidentialFeedName = "external_residential.txt"

func externalResidentialFeedPath(dir string) string {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		dir = "/var/lib/ad-event-processor/proxy-vpn"
	}
	return filepath.Join(dir, externalResidentialFeedName)
}

func residentialIntelFeedLine(ip string) (string, bool) {
	addr, err := netip.ParseAddr(strings.TrimSpace(ip))
	if err != nil || !addr.IsValid() {
		return "", false
	}
	bits := addr.BitLen()
	prefix := netip.PrefixFrom(addr, bits).Masked()
	if !prefix.IsValid() {
		return "", false
	}
	return fmt.Sprintf("%s 0 vpn", prefix.String()), true
}

func appendResidentialIntelFeed(dir, ip string) error {
	line, ok := residentialIntelFeedLine(ip)
	if !ok {
		return ErrInvalidIP
	}
	path := externalResidentialFeedPath(dir)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir residential intel feed dir: %w", err)
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open residential intel feed: %w", err)
	}
	defer func() { _ = f.Close() }()
	if _, err := f.WriteString(line + "\n"); err != nil {
		return fmt.Errorf("append residential intel feed: %w", err)
	}
	return nil
}
