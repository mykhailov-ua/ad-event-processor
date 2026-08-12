package doctor

// CheckHint returns a one-line remediation command for a failed or warned check ID.
func CheckHint(id string) string {
	if hint, ok := checkHints[id]; ok {
		return hint
	}
	return ""
}

var checkHints = map[string]string{
	"redis":      "docker compose logs redis-0 redis-1 redis-2 redis-3",
	"clickhouse": "docker compose logs clickhouse",
	"dns":        "point tracking domain A-record to this host; configure nginx/Caddy for TLS",
	"kernel":     "optional for appliance; edge XDP needs kernel >= 6.1 with BTF",
	"sysctl":     "raise net.core.somaxconn (see internal/installer preflight)",
	"listen":     "see docs/EDGE_CASES.md §3 and §8; raise somaxconn and recreate listeners after sysctl changes",
	"disk":       "df -h && docker system df",
	"tls":        "set DB_DSN sslmode=verify-full for production Postgres TLS",
	"rtb_config": "review RTB settings in control UI or GET /api/v1/settings/platform",
	"license":    "apply monthly JWT: Settings → License or POST /api/v1/license/apply",
	"edge_xdp":   "enable Enterprise license (ebpf_xdp_edge), BTF kernel, installer systemd units; bash scripts/install/bidshard-install.sh apply",
	"slotmap":    "see docs/TRADEOFFS.md §1; reload nginx to re-sync edge-slot-map.lua from GET /ops/shards/slot-map",
}
