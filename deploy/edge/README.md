# Edge deploy (XDP + host tuning)

Enterprise perimeter extension — **not** part of the default `single_vps` appliance SKU. Default perimeter remains Nginx OpenResty Lua (`deploy/nginx/lua/access_check.lua`).

Operator overview: [docs/XDP.md](../../docs/XDP.md).

---

## Layout

| Path | Purpose |
| :--- | :--- |
| `xdp/bpf/edge_filter.c` | XDP L4 filter (blocklist, allowlist, SYN/PPS limits, ringbufs) |
| `xdp/Dockerfile` | Lab image: `edge-xdp` + `edge-bpf-sync` + `entrypoint.sh` |
| `xdp/entrypoint.sh` | Container supervisor: start `edge-xdp`, wait for pinned maps, exec `edge-bpf-sync` |
| `nic-tune.service` | Oneshot systemd unit for RX ring / IRQ affinity |
| `nic-tune.env.example` | `INGRESS_INTERFACE`, `IRQ_STRATEGY` for nic-tune |
| `99-ad-event-processor-sysctl.conf` | Host sysctl baseline for edge ingress |
| `99-espx-edge.conf` | Legacy sysctl filename (same role; prefer `99-ad-event-processor-sysctl.conf` on new installs) |

Binaries: `cmd/edge-xdp` (attach + pin maps), `cmd/edge-bpf-sync` (Redis → BPF maps).

---

## BPF pin directory

Canonical pin root: **`BPF_PIN_DIR`** (default `/sys/fs/bpf/ad-event-processor`).

`edge-xdp` creates the directory and pins all maps before XDP attach. `edge-bpf-sync` opens maps under the same root (or explicit `BPF_*_MAP` overrides).

| Map file | Role |
| :--- | :--- |
| `blocklist_v4` | LPM deny list (manual, auto, fraud) |
| `allow_v4` | Partner allowlist |
| `syn_ratelimit_v4` | Per-source SYN token bucket |
| `syn_subnet_ratelimit_v4` | /24 SYN cap |
| `ratelimit_v4` | Per-source PPS bucket |
| `rst_ratelimit_v4` | RST rate limit state |
| `global_syn` | Global SYN counter |
| `stats` | Per-CPU drop/pass counters (Prometheus + Redis snapshot) |
| `config` | Runtime knobs (SYN cookie, fingerprint enable) |
| `violations` | Ringbuf → autoban → resync |
| `fingerprints` | Ringbuf → Redis IVT staging |
| `prog_array` | XDP program dispatch |

Smoke (paths + pin/open alignment):

```bash
bash scripts/test/edge_pin_dir_smoke.sh
```

---

## `edge-xdp`

Attaches the XDP program to the ingress NIC and pins maps. Requires **CAP_BPF**, **CAP_NET_ADMIN**, BTF-enabled kernel (6.1+), and Enterprise license (`ebpf_xdp_edge` in Redis `entitlement:deployment`). Without entitlement, `edge-xdp` pins maps but skips attach; `edge-bpf-sync` idles.

### CLI flags

| Flag | Env fallback | Default | Description |
| :--- | :--- | :--- | :--- |
| `-iface` | `INGRESS_INTERFACE` | *(required)* | NIC name (`eth0`, `ens3`, …) |
| `-pin-dir` | `BPF_PIN_DIR` | `/sys/fs/bpf/ad-event-processor` | Pin directory for all maps |
| `-mode` | `XDP_MODE` | `generic` | `generic`, `native`, or `offload` |

### Runtime config (BPF `config` map)

Set before or at start; read by `InitConfigFromEnv`:

| Variable | Default | Effect |
| :--- | :--- | :--- |
| `XDP_SYN_COOKIE` | off | `1` or `true` enables SYN cookies |
| `XDP_FINGERPRINT` | on | `0` or `false` disables passive TCP fingerprint ringbuf |

### Manual attach (lab)

```bash
make bpf-dev   # or: go generate in deploy/edge/xdp
go build -o bin/edge-xdp ./cmd/edge-xdp

export INGRESS_INTERFACE=eth0
export BPF_PIN_DIR=/sys/fs/bpf/ad-event-processor
export XDP_MODE=generic

sudo -E ./bin/edge-xdp
# equivalent:
sudo ./bin/edge-xdp -iface eth0 -pin-dir /sys/fs/bpf/ad-event-processor -mode generic
```

Verify pinned blocklist:

```bash
ls -l "${BPF_PIN_DIR:-/sys/fs/bpf/ad-event-processor}/blocklist_v4"
bpftool map show pinned "${BPF_PIN_DIR:-/sys/fs/bpf/ad-event-processor}/blocklist_v4"
```

### Installer systemd

When `install.yaml` has `edge_xdp: true` (and BTF preflight passes), the installer renders and enables:

- `ad-event-processor-edge-xdp.service` — `INGRESS_INTERFACE`, `BPF_PIN_DIR`
- `ad-event-processor-edge-bpf-sync.service` — `BPF_PIN_DIR`, `EnvironmentFile` secrets (`REDIS_ADDRS`, …)

---

## `edge-bpf-sync`

Syncs Redis shard-0 blacklist/allowlist keys into pinned maps; drains violation and fingerprint ringbufs. **Must start after** maps are pinned (systemd `Requires=` edge-xdp unit, or `entrypoint.sh` wait loop in Docker).

### Environment

| Variable | Default | Description |
| :--- | :--- | :--- |
| `BPF_PIN_DIR` | `/sys/fs/bpf/ad-event-processor` | Pin root; used when `BPF_*_MAP` unset |
| `BPF_BLOCKLIST_MAP` | `$BPF_PIN_DIR/blocklist_v4` | Override blocklist pin path |
| `BPF_ALLOWLIST_MAP` | `$BPF_PIN_DIR/allow_v4` | Override allowlist pin path |
| `BPF_STATS_MAP` | `$BPF_PIN_DIR/stats` | Stats map (metrics optional if missing) |
| `BPF_VIOLATIONS_MAP` | `$BPF_PIN_DIR/violations` | Violation ringbuf |
| `BPF_FINGERPRINTS_MAP` | `$BPF_PIN_DIR/fingerprints` | Fingerprint ringbuf |
| `SYNC_INTERVAL` | `5s` | Redis → BPF map poll interval |
| `STATS_INTERVAL` | `2s` | Stats export to Prometheus + Redis |
| `VIOLATION_POLL_INTERVAL` | `250ms` | Violation ringbuf drain period |
| `FINGERPRINT_POLL_INTERVAL` | `500ms` | Fingerprint ringbuf drain period |
| `AUTOBAN_TTL` | `5m` | TTL for `blacklist:auto` after violation |
| `METRICS_PORT` | `9090` | Prometheus `/metrics` listen port |
| `REDIS_ADDRS` or `REDIS_HOST`/`REDIS_PORT` | — | **Required** — first addr used (shard 0) |
| `REDIS_PASS` | — | Redis password |

`XDP_SYN_COOKIE` and `XDP_FINGERPRINT` apply to **`edge-xdp`** only (config map at attach). bpf-sync does not re-read them.

### Manual start (after edge-xdp)

```bash
go build -o bin/edge-bpf-sync ./cmd/edge-bpf-sync
export BPF_PIN_DIR=/sys/fs/bpf/ad-event-processor
export REDIS_ADDRS=127.0.0.1:6379
./bin/edge-bpf-sync
```

---

## Docker (`deploy/edge/xdp`)

Image entrypoint exports `BPF_PIN_DIR` (default `/sys/fs/bpf/ad-event-processor`), starts `edge-xdp`, waits for `${BPF_PIN_DIR}/blocklist_v4`, then execs `edge-bpf-sync`. Requires host `/sys/fs/bpf`, BTF, `privileged`, and `network_mode: host`.

**Compose lab profile** (not `single_vps` / `compose_dev`):

```bash
# .env: EDGE_XDP_INGRESS_INTERFACE=lo (lab) or eth0 (ingress NIC)
docker compose --profile enterprise-xdp up -d redis-0 edge-xdp
bash scripts/test/edge_xdp_compose_smoke.sh   # BTF host: attach + sync smoke
```

Constraints: `privileged: true`, `network_mode: host`, `pid: host`, `seccomp:unconfined`, mounts `/sys/fs/bpf` and `/sys/kernel/btf`. On kernels where `xdp_syn_cookie` fails verifier load, `edge-xdp` retries without that program (SYN cookies unavailable until portability work lands).

---

## NIC tuning (optional)

RX ring and IRQ affinity for high PPS ingress. Independent of XDP attach but should target the same NIC as `INGRESS_INTERFACE`.

```bash
# Install unit + helper (root)
sudo bash scripts/ops/nic_tune.sh install-systemd
# Edit /etc/ad-event-processor/edge-nic-tune.env if needed
sudo systemctl start espx-edge-nic-tune.service
```

Files: `deploy/edge/nic-tune.service`, `deploy/edge/nic-tune.env.example`. Installed helper: `/usr/local/bin/edge_nic_tune.sh`.

---

## Host sysctl (optional)

```bash
sudo cp deploy/edge/99-ad-event-processor-sysctl.conf /etc/sysctl.d/
sudo sysctl --system
```

Or use `scripts/ops/sysctl.sh` with `deploy/edge/99-espx-edge.conf` on legacy hosts.

---

## Rollback

1. **Stop sync, then detach XDP** (order matters — bpf-sync holds map fds):

   ```bash
   sudo systemctl stop ad-event-processor-edge-bpf-sync.service
   sudo systemctl stop ad-event-processor-edge-xdp.service
   ```

   Or kill container / manual `edge-xdp` process (SIGTERM detaches via defer).

2. **Nginx Lua blacklist remains active** — perimeter does not go open; only kernel-fast drop is removed.

3. **Disable on profile change**: installer `edge_xdp: false` stops and disables both units (`render_edge_systemd.go`).

4. **Pinned maps**: stale pins under `BPF_PIN_DIR` are replaced on next `edge-xdp` start (`os.Remove` before pin). Safe to remove the directory when both services are stopped.

No tracker or processor restart required.

---

## Verification

| Check | Command |
| :--- | :--- |
| Compliance (no ebpf in hot path) | `bash scripts/ci/compliance.sh` |
| Pin-dir alignment | `bash scripts/test/edge_pin_dir_smoke.sh` |
| BPF unit tests | `go test ./internal/edge/... -count=1` |
| BTF present | `test -r /sys/kernel/btf/vmlinux && echo ok` |
| Sync lag | `edge-bpf-sync` logs; Redis `blacklist:*` keys; ops XDP panel when snapshot present |

Bench reference (lab): `go test ./internal/edge/ -bench='BenchmarkXDP_' -benchmem` (harness `xdp_prog_test`).

---

## Tier D — CO-RE portability and NIC offload (Enterprise SOW)

Tier A–C (blocklist, SYN/PPS limits, ringbufs) ship in `edge_filter.c` and are validated in **generic** XDP mode. Tier D covers portable builds across LTS kernels and optional hardware offload — **not** appliance pilot SLA.

### CO-RE / BTF build path

| Step | Command / requirement |
| :--- | :--- |
| Host kernel | **6.1+ LTS** with `CONFIG_DEBUG_INFO_BTF=y` (`/sys/kernel/btf/vmlinux` readable) |
| Build deps | `clang`, `llvm`, `libbpf-dev`, `linux-libc-dev` (see `deploy/edge/xdp/Dockerfile` builder stage) |
| Generate + compile | `go generate ./internal/edge/` (bpf2go embeds `edge_filter.c` for `bpfel`/`bpfeb`) |
| Local dev | `make bpf-dev` then `go build -o bin/edge-xdp ./cmd/edge-xdp` |
| Loader | cilium/ebpf attaches using host BTF; no separate `.o` deploy on appliance (objects embedded in binary) |

Installer preflight: `PF-BTF` in `internal/installer/preflight_linux.go` when `edge_xdp: true`.

Supported LTS examples (lab): Ubuntu 22.04/24.04 HWE, Debian 12 bookworm — kernel **≥ 6.1** with BTF enabled. Custom kernels without BTF fail load at `edge-xdp` startup.

### `XDP_MODE` attach modes

| Mode | Use case | Notes |
| :--- | :--- | :--- |
| `generic` | **Default** — compliance benches, compose lab, VMs | SKB path; portable; Tier A–C SLA baseline |
| `native` | Bare-metal ingress NIC | Driver XDP hook; lower latency when supported |
| `offload` | SmartNIC / HW offload lab only | Requires driver offload support; often unavailable on cloud VMs |

`edge-xdp` **auto-fallback**: `offload` → `native` → `generic`; `native` → `generic`. Logs `xdp attach fell back to slower mode` when a faster mode fails.

```bash
# Lab only — appliance installer defaults to generic
export XDP_MODE=offload   # falls back if NIC lacks offload
sudo -E ./bin/edge-xdp -iface eth0 -mode offload
```

Do **not** set `offload` on appliance pilot SKU; use `generic` unless Enterprise SOW specifies NIC tuning.

### Verification (Tier D lab)

Perf: `BenchmarkXDP_*` = prog test only (harness `xdp_prog_test`); kernel proof = `edge-xdp-fault` / `scripts/test/xdp_resilience_drill.sh` — see [XDP.md](../../docs/XDP.md) Tier D lab verification.

```bash
bash scripts/ci/compliance.sh
go test ./internal/edge/... -count=1
bash scripts/test/edge_xdp_bench_gate.sh   # skips without BTF
```

Compliance and `BenchmarkXDP_*` baselines use **generic** userspace program tests (`internal/edge/bench_test.go`, harness `xdp_prog_test`) — not kernel RX; unchanged by offload path.

### Known detection limits

XDP is defense-in-depth only. Low-volume /24 rotation and fingerprint-off/congested paths remain accepted limits — backstop: Lua rate limit, tracker filters, IVT, fraud-scorer. Details: [XDP.md](../../docs/XDP.md) Known detection limits.

### Lab fault injector

Enterprise lab only — **not** installed on appliance default (`edge_xdp: false`).

| Tool | Purpose |
| :--- | :--- |
| `cmd/edge-xdp-fault` | Malformed + high-rate SYN injection (`program` = userspace bpf.Test; `iface` = raw socket, needs attached XDP on NIC) |
| `scripts/fault/xdp_injector_drill.sh` | Smoke drill; prints `fault_proof fault=xdp_injector_*` lines |

```bash
go build -o bin/edge-xdp-fault ./cmd/edge-xdp-fault
./bin/edge-xdp-fault -mode program          # userspace BPF; no NIC attach required
sudo ./bin/edge-xdp-fault -mode iface -iface eth0 -dst 10.0.0.1   # raw socket; edge-xdp must be attached on eth0
bash scripts/fault/xdp_injector_drill.sh
```

`internal/edge/*_fault_test.go` also emit `fault_proof fault=xdp_*` for `scripts/fault/run.sh` when `./internal/edge/...` is included.
