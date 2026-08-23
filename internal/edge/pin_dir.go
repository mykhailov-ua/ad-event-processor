package edge

import (
	"os"
	"path/filepath"
)

const DefaultBPFPinDir = "/sys/fs/bpf/ad-event-processor"

const (
	MapBlocklistV4     = "blocklist_v4"
	MapBlocklistV6     = "blocklist_v6"
	MapBlocklistHostV4 = "blocklist_host_v4"
	MapBlocklistHostV6 = "blocklist_host_v6"
	MapAllowV4         = "allow_v4"
	MapAllowV6      = "allow_v6"
	MapStats        = "stats"
	MapViolations   = "violations"
	MapFingerprints = "fingerprints"
)

const (
	DefaultBlocklistMapPath    = DefaultBPFPinDir + "/" + MapBlocklistV4
	DefaultBlocklistV6MapPath  = DefaultBPFPinDir + "/" + MapBlocklistV6
	DefaultAllowlistMapPath    = DefaultBPFPinDir + "/" + MapAllowV4
	DefaultAllowlistV6MapPath  = DefaultBPFPinDir + "/" + MapAllowV6
	DefaultStatsMapPath        = DefaultBPFPinDir + "/" + MapStats
	DefaultViolationsMapPath   = DefaultBPFPinDir + "/" + MapViolations
	DefaultFingerprintsMapPath = DefaultBPFPinDir + "/" + MapFingerprints
)

func BPFPinDir() string {
	return EnvOr("BPF_PIN_DIR", DefaultBPFPinDir)
}

func PinnedMapPath(pinDir, mapName string) string {
	return filepath.Join(pinDir, mapName)
}

type PinnedMapPaths struct {
	Blocklist       string
	BlocklistV6     string
	BlocklistHostV4 string
	BlocklistHostV6 string
	Allowlist       string
	AllowlistV6     string
	Stats           string
	Violations      string
	Fingerprints    string
}

func ResolvePinnedMapPaths() PinnedMapPaths {
	pinDir := BPFPinDir()
	return PinnedMapPaths{
		Blocklist:       envOrPinnedMap("BPF_BLOCKLIST_MAP", pinDir, MapBlocklistV4),
		BlocklistV6:     envOrPinnedMap("BPF_BLOCKLIST_V6_MAP", pinDir, MapBlocklistV6),
		BlocklistHostV4: envOrPinnedMap("BPF_BLOCKLIST_HOST_V4_MAP", pinDir, MapBlocklistHostV4),
		BlocklistHostV6: envOrPinnedMap("BPF_BLOCKLIST_HOST_V6_MAP", pinDir, MapBlocklistHostV6),
		Allowlist:       envOrPinnedMap("BPF_ALLOWLIST_MAP", pinDir, MapAllowV4),
		AllowlistV6:     envOrPinnedMap("BPF_ALLOWLIST_V6_MAP", pinDir, MapAllowV6),
		Stats:           envOrPinnedMap("BPF_STATS_MAP", pinDir, MapStats),
		Violations:      envOrPinnedMap("BPF_VIOLATIONS_MAP", pinDir, MapViolations),
		Fingerprints:    envOrPinnedMap("BPF_FINGERPRINTS_MAP", pinDir, MapFingerprints),
	}
}

func envOrPinnedMap(envKey, pinDir, mapName string) string {
	if v := os.Getenv(envKey); v != "" {
		return v
	}
	return PinnedMapPath(pinDir, mapName)
}
