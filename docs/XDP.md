# XDP edge filtering (Enterprise)

**Not appliance SKU.** Default perimeter: Nginx Lua (`deploy/nginx/lua/access_check.lua`). XDP drops blacklisted IPs at NIC before userspace. No HTTP parse — out of scope for [PARSER.md](PARSER.md). Pilot: `ebpf_xdp_edge: false`.

Use when: high PPS floods, kernel-fast drop contract, BTF kernel 6.1+ ingress NIC.

## Enable

| Step | Action |
| :--- | :--- |
| License | `features.ebpf_xdp_edge: true` |
| Installer | `edge_xdp: true` (BTF preflight) |
| Build | `make bpf-dev` / `make gen` |
| Binaries | `edge-xdp` (attach + pin), `edge-bpf-sync` (Redis → maps) |

Without entitlement: pins maps, skips attach; bpf-sync idles. Lab compose: `docker compose --profile enterprise-xdp`.

**Flow:** control Outbox → Redis shard 0 → bpf-sync → BPF maps → edge-xdp drop → Nginx → tracker. Hot path does not import `internal/edge`.

## Pin directory (`BPF_PIN_DIR`)

Default `/sys/fs/bpf/ad-event-processor`. Key maps: `blocklist_v4`, `blocklist_v6`, `allow_v4`, `allow_v6`, `syn_ratelimit_v4`, `syn_subnet_ratelimit_v4`, `ratelimit_v4`, `stats`, `config`, `violations`, `fingerprints`.

IPv6 deny/allow use `BPF_MAP_TYPE_LPM_TRIE` with `BPF_F_NO_PREALLOC` (`blocklist_v6`, `allow_v6`). bpf-sync applies Redis sets to both v4 host keys and v6 host keys; allowlist lookup runs before blocklist on IPv6 TCP to tracker ingress.

## `edge-xdp`

CAP_BPF + CAP_NET_ADMIN. Flags: `-iface`/`INGRESS_INTERFACE`, `-pin-dir`/`BPF_PIN_DIR`, `-mode`/`XDP_MODE` (`generic`|`native`|`offload`, default `generic`). Config map: `XDP_SYN_COOKIE` (off), `XDP_FINGERPRINT` (on).

```bash
make bpf-dev && go build -o bin/edge-xdp ./cmd/edge-xdp
export INGRESS_INTERFACE=eth0 BPF_PIN_DIR=/sys/fs/bpf/ad-event-processor
sudo -E ./bin/edge-xdp
```

Installer units: `ad-event-processor-edge-xdp.service`, `ad-event-processor-edge-bpf-sync.service`.

## `edge-bpf-sync`

Start after maps pinned. Required: `REDIS_ADDRS` (shard 0).

| Variable | Default |
| :--- | :--- |
| `SYNC_INTERVAL` | `5s` |
| `STATS_INTERVAL` | `2s` |
| `VIOLATION_POLL_INTERVAL` | `250ms` |
| `FINGERPRINT_POLL_INTERVAL` | `500ms` |
| `AUTOBAN_TTL` | `5m` |
| `METRICS_PORT` | `9090` |
| `BPF_*_MAP` | under `BPF_PIN_DIR` |

```bash
go build -o bin/edge-bpf-sync ./cmd/edge-bpf-sync
export BPF_PIN_DIR=/sys/fs/bpf/ad-event-processor REDIS_ADDRS=127.0.0.1:6379
./bin/edge-bpf-sync
```

## Rollback

Stop bpf-sync → stop edge-xdp. Nginx Lua blacklist stays active. No tracker restart. Stale pins replaced on next start.

## Verification

```bash
bash scripts/ci/compliance.sh
bash scripts/test/edge_pin_dir_smoke.sh
go test ./internal/edge/... -count=1
```

Bench (lab, prog test only): `go test ./internal/edge/ -bench='BenchmarkXDP_' -benchmem`.

## Tier D (CO-RE / offload)

Build: kernel 6.1+ BTF, `go generate ./internal/edge/`. Modes auto-fallback: `offload` → `native` → `generic`. Lab fault: `cmd/edge-xdp-fault`, `bash scripts/fault/xdp_injector_drill.sh`.

## Detection limits

| ID | Limit | Backstop |
| :--- | :--- | :--- |
| X-06 | Low-volume /24 rotation under subnet cap | Lua limits; FilterEngine; IVT |
| X-07 | Fingerprint ringbuf zero when off/congested | IVT; Lua blacklist |

High-volume /24 bursts dropped at cap. Optional NIC tune: `scripts/ops/nic_tune.sh install-systemd`; sysctl: `deploy/edge/99-ad-event-processor-sysctl.conf`.

## Troubleshooting

| Symptom | Check |
| :--- | :--- |
| Load fail | `/sys/kernel/btf/vmlinux`; kernel ≥ 6.1 |
| Stale list | bpf-sync logs; Redis shard 0 |
| False drops | Lua list vs BPF map lag |
