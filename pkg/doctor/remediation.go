package doctor

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
	"listen":     "raise somaxconn and recreate listeners after sysctl changes (deploy/edge/99-ad-event-processor-sysctl.conf; bash scripts/ops/sysctl.sh)",
	"disk":       "df -h && docker system df",
	"tls":        "set DB_DSN sslmode=verify-full for production Postgres TLS",
	"rtb_config": "review RTB settings in control UI or GET /api/v1/settings/platform",
	"license":    "apply monthly JWT: Settings -> License or POST /api/v1/license/apply",
	"edge_xdp":   "enable Enterprise license (ebpf_xdp_edge), BTF kernel, installer systemd units; bash scripts/install/ad-event-processor-install.sh apply",
	"slotmap":    "reload nginx to re-sync edge-slot-map.lua from GET /api/v1/ops/shards/slot-map (docs/ARCHITECTURE.md section 4.1)",
}
