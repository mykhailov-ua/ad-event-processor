package licensing

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"

	"espx/internal/config"
)

func HostFingerprint() string {
	machineID := readMachineID()
	paths := stableInstallPaths()
	h := sha256.New()
	_, _ = h.Write([]byte(machineID))
	for _, p := range paths {
		_, _ = h.Write([]byte{0})
		_, _ = h.Write([]byte(p))
	}
	return hex.EncodeToString(h.Sum(nil))
}

func readMachineID() string {
	for _, path := range []string{"/etc/machine-id", "/var/lib/dbus/machine-id"} {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		id := strings.TrimSpace(string(data))
		if id != "" {
			return id
		}
	}
	return ""
}

func stableInstallPaths() []string {
	var paths []string
	if root := strings.TrimSpace(os.Getenv("ROOT")); root != "" {
		paths = append(paths, filepath.Clean(root))
	}
	if cwd, err := os.Getwd(); err == nil {
		paths = append(paths, filepath.Clean(cwd))
	}
	if p := strings.TrimSpace(config.LicenseEnv("PATH")); p != "" {
		paths = append(paths, filepath.Dir(filepath.Clean(p)))
	}
	seen := make(map[string]struct{}, len(paths))
	out := make([]string, 0, len(paths))
	for _, p := range paths {
		if p == "" {
			continue
		}
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	return out
}
