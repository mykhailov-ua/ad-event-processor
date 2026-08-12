# XDP edge filtering (Enterprise)

**Not part of appliance SKU.** Default perimeter is **Nginx OpenResty Lua** (`deploy/nginx/lua/access_check.lua`). XDP adds optional NIC-level drop for blacklisted IPs before traffic reaches userspace.

**Parser security boundary:** XDP is **out of scope** for [PARSER_SECURITY.md](../PARSER_SECURITY.md) (no HTTP/JSON parse). See §9.3 there and milestone `.cursor/PARSER_SECURITY_MILESTONE.md` §7.

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
| 5. Config | `INGRESS_INTERFACE` (NIC name), map paths per `deploy/edge/` README |

`edge-bpf-sync` idles without entitlement (`cmd/edge-bpf-sync`).

XDP is **not** a service in default `deploy/compose/docker-compose.yaml` (`single_vps`).

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
sudo ./bin/edge-xdp -iface eth0 -map /var/lib/ad-event-processor/xdp/blacklist.bin
```

Use installer-generated systemd units when deployed via `ad-event-processor-install.sh` with `edge_xdp=true`.

### Verify

- Compliance rules: `bash scripts/ci/compliance.sh` (defensive XDP fingerprint; does not require loaded program for CI merge).
- Bench reference: [BENCHMARKS.md](../BENCHMARKS.md) §A.10 (dev/lab only).

### Rollback

Detach XDP program; Nginx Lua blacklist remains active. No tracker restart required.

---

## Troubleshooting

| Symptom | Check |
| :--- | :--- |
| Program will not load | `bpftool btf dump file /sys/kernel/btf/vmlinux`; kernel ≥ 6.1 |
| Stale blacklist | `edge-bpf-sync` logs; Redis shard 0 connectivity |
| Legitimate traffic dropped | Compare Lua edge list vs BPF map; sync lag |
