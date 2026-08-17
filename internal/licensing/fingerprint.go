package licensing

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"

	"github.com/bidshard/ad-event-processor/internal/config"
	"github.com/bidshard/ad-event-processor/pkg/coldpath"
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
	return coldpath.UniqueSlice(paths)
}
