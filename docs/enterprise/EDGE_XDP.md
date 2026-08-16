# XDP edge filtering (Enterprise)

**Not part of appliance SKU.** Default perimeter is **Nginx OpenResty Lua** (`deploy/nginx/lua/access_check.lua`). XDP adds optional NIC-level drop for blacklisted IPs before traffic reaches userspace.

**Parser security boundary:** XDP is **out of scope** for [PARSER_SECURITY.md](../PARSER_SECURITY.md) (no HTTP/JSON parse). See §9.3 there; closed milestone [MILESTONES.md](../MILESTONES.md) §2.

See: [FROZEN_FEATURES.md](../FROZEN_FEATURES.md), [ARCHITECTURE.md](../ARCHITECTURE.md) §1.

---

## When to use

- High-volume DDoS or bot floods where Lua/userspace cost is measurable.
- Contract requires kernel-fast drop with synced blacklist maps.
- Host has **BTF-enabled kernel 6.1+** and dedicated ingress NIC.

Appliance pilot: `ebpf_xdp_edge: false` in `deploy/vendor/sku.yaml`.

---

## Enable path

| Step | Action |
| :--- | :--- |
| 1. License | JWT `features.ebpf_xdp_edge: true` (Enterprise SKU in `deploy/vendor/sku.yaml`) |
| 2. Installer | `edge_xdp: true` in install profile; `preflight_linux.go` checks BTF |
| 3. Build BPF | `make bpf-dev` or `make gen` — objects under `deploy/edge/xdp/` |
| 4. Binaries | `edge-xdp` (attach XDP), `edge-bpf-sync` (outbox → Redis shard 0 → BPF maps) |
| 5. Config | `INGRESS_INTERFACE`, `BPF_PIN_DIR`, env knobs — [deploy/edge/README.md](../../deploy/edge/README.md) |

`edge-bpf-sync` idles without entitlement (`cmd/edge-bpf-sync`). `edge-xdp` pins maps but skips NIC attach when `ebpf_xdp_edge` is `0`.

XDP is **not** a service in default `deploy/compose/docker-compose.yaml` (`single_vps`). Optional lab profile: `docker compose --profile enterprise-xdp` — see [deploy/edge/README.md](../../deploy/edge/README.md).

---

## Data flow

1. `control` OutboxWorker publishes blacklist updates to Redis shard 0.
2. `edge-bpf-sync` reads outbox/Redis and updates BPF hash maps.
3. `edge-xdp` attached to ingress NIC drops packets for blacklisted IPs (pass otherwise).
4. Clean traffic continues to Nginx → tracker as in appliance diagram.

Hot path (`tracker`, `processor`) does **not** import `internal/edge/bpf` — isolation via separate `cmd/` binaries only.

---

## Operations

### Attach (example)

```bash
# After license + BPF build on host
make bpf-dev
go build -o bin/edge-xdp ./cmd/edge-xdp

export INGRESS_INTERFACE=eth0
export BPF_PIN_DIR=/sys/fs/bpf/ad-event-processor

sudo -E ./bin/edge-xdp
# or explicit flags:
sudo ./bin/edge-xdp -iface eth0 -pin-dir /sys/fs/bpf/ad-event-processor -mode generic
```

Map layout, bpf-sync env, nic-tune, and rollback: [deploy/edge/README.md](../../deploy/edge/README.md).

Use installer-generated systemd units when deployed via `ad-event-processor-install.sh` with `edge_xdp: true`.

### `edge-bpf-sync` environment (summary)

| Variable | Default | Role |
| :--- | :--- | :--- |
| `SYNC_INTERVAL` | `5s` | Redis → BPF map sync |
| `BPF_PIN_DIR` | `/sys/fs/bpf/ad-event-processor` | Pin root when overrides unset |
| `BPF_BLOCKLIST_MAP` | `$BPF_PIN_DIR/blocklist_v4` | Blocklist map path |
| `BPF_ALLOWLIST_MAP` | `$BPF_PIN_DIR/allow_v4` | Allowlist map path |
| `BPF_STATS_MAP` | `$BPF_PIN_DIR/stats` | Stats map |
| `BPF_VIOLATIONS_MAP` | `$BPF_PIN_DIR/violations` | Violation ringbuf |
| `BPF_FINGERPRINTS_MAP` | `$BPF_PIN_DIR/fingerprints` | Fingerprint ringbuf |
| `XDP_SYN_COOKIE` | off | Set on **edge-xdp** (`1`/`true` enables) |
| `XDP_FINGERPRINT` | on | Set on **edge-xdp** (`0`/`false` disables) |

Full list (`STATS_INTERVAL`, `REDIS_ADDRS`, …): [deploy/edge/README.md](../../deploy/edge/README.md).

### Verify

- Compliance rules: `bash scripts/ci/compliance.sh` (defensive XDP fingerprint; does not require loaded program for CI merge).
- Bench reference: [BENCHMARKS.md](../BENCHMARKS.md) §A.10 (dev/lab only).

### Rollback

Stop `edge-bpf-sync`, then `edge-xdp` (or disable installer units). Detach removes XDP; **Nginx Lua blacklist remains active**. No tracker restart required. Steps: [deploy/edge/README.md](../../deploy/edge/README.md) §Rollback.

---

## Tier D — CO-RE portability and NIC offload (Enterprise SOW)

Not appliance pilot scope. Tier A–C behavior is validated in **generic** XDP mode (`XDP_MODE=generic`, default).

### CO-RE / BTF build

- **Host:** kernel 6.1+ LTS with BTF (`/sys/kernel/btf/vmlinux`).
- **Build:** `go generate ./internal/edge/bpf/` (bpf2go + clang); local shortcut `make bpf-dev`.
- **Runtime:** embedded BPF objects loaded via cilium/ebpf against host BTF — no separate `.o` on disk for appliance units.

Details: [deploy/edge/README.md](../../deploy/edge/README.md) §Tier D.

### `XDP_MODE` and offload fallback

| Mode | Role |
| :--- | :--- |
| `generic` | Default; compliance benches; VMs and lab compose |
| `native` | Bare-metal driver XDP when supported |
| `offload` | SmartNIC lab only — `edge-xdp` falls back to `native` then `generic` on attach failure |

```bash
export XDP_MODE=offload   # lab; auto-fallback if HW offload unavailable
sudo ./bin/edge-xdp -iface eth0 -mode offload
```

### Lab verification

Perf: `BenchmarkXDP_*` = **prog test only** (harness `xdp_prog_test`; userspace `prog.Run` in `internal/edge/bpf/bench_test.go`). Not kernel NIC RX or pinned-attach drop rates. Kernel drop proof: `cmd/edge-xdp-fault`, `scripts/test/xdp_resilience_drill.sh`, pinned attach drills on lab NIC.

```bash
bash scripts/ci/compliance.sh
bash scripts/test/edge_xdp_bench_gate.sh
go test -run='^$' -bench='BenchmarkXDP_' -benchmem ./internal/edge/bpf/ -count=5
```

---

## Known detection limits (defense in depth)

XDP is **not** the sole perimeter. Nginx Lua (`access_check.lua`), tracker `FilterEngine`, IVT detector, and `fraud-scorer` remain authoritative for L7 and billing risk.

| ID | Limit | Decision | Backstop |
| :--- | :--- | :--- | :--- |
| **X-06** | Low-volume /24 rotation (≤1 SYN per host, spread across a subnet) may stay under default `syn_subnet_limit` (256/window) | **Accepted** — no default tuning | Lua per-IP/campaign rate limits; tracker filters; IVT + fraud-scorer |
| **X-07** | Fingerprint ringbuf may emit zero when `XDP_FINGERPRINT=0` or under ringbuf congestion | **Accepted** — fingerprint is best-effort telemetry | IVT pipeline uses other signals; Lua blacklist + manual bans |

High-volume /24 bursts **are** dropped when the subnet cap is exceeded (see `TestXDP_dropSynSubnetFlood`, `TestFraudScenarios_X06_HighVolumeSubnetBurstDrops`). Operators may lower `syn_subnet_limit` via BPF `config` map for stricter Enterprise contracts — re-run `BenchmarkXDP_*` (harness `xdp_prog_test`) after changes.

Scenario corpus: `internal/edge/bpf/edge_filter_fraud_scenarios_test.go`.

---

## Troubleshooting

| Symptom | Check |
| :--- | :--- |
| Program will not load | `bpftool btf dump file /sys/kernel/btf/vmlinux`; kernel ≥ 6.1 |
| Stale blacklist | `edge-bpf-sync` logs; Redis shard 0 connectivity |
| Legitimate traffic dropped | Compare Lua edge list vs BPF map; sync lag |
