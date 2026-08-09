# Frozen features (Enterprise / SOW)

Appliance SKU (`single_vps`, pilot license) ships **without** multi-region proxy and NIC-level XDP. Code, fault tests, and compose profiles remain in git for Enterprise contracts.

Canonical policy: [CUT_CANDIDATES.md](CUT_CANDIDATES.md) §2, milestone `.cursor/MILESTONE.md` §7.

---

## Enable path summary

| Feature | License JWT (`features.*`) | Installer profile | Compose | Operator doc |
| :--- | :--- | :--- | :--- | :--- |
| **Multi-region / `region-proxy`** | `multi_region: true` | `multi_region: true` (blocked in `compose_dev`) | `--profile multi-region` | [enterprise/MULTI_REGION.md](enterprise/MULTI_REGION.md) |
| **XDP edge (`edge-xdp`)** | `ebpf_xdp_edge: true` | `edge_xdp: true` (BTF preflight) | Not in `single_vps`; manual systemd from installer | [enterprise/EDGE_XDP.md](enterprise/EDGE_XDP.md) |

Pilot defaults: `deploy/vendor/sku.yaml` — both features `false`.

Runtime checks:

- `control` refuses start when `MULTI_REGION_ENABLED=1` without `multi_region` entitlement (`internal/controlplane/serve.go`).
- `edge-bpf-sync` idles without `ebpf_xdp_edge` (`cmd/edge-bpf-sync`).
- Release pilot images exclude `region-proxy` binary (`.github/workflows/release-images.yaml`).

---

## What stays in appliance default

| Layer | Default perimeter |
| :--- | :--- |
| Edge filtering | Nginx OpenResty Lua (`deploy/nginx/lua/`) — blacklist, rate limit, shard pick |
| Redis HA lab | Sentinel overlay (`deploy/compose/docker-compose.sentinel.yaml`) — **not** product multi-region; see [EDGE_CASES.md](EDGE_CASES.md) §10 |
| Hot path | Tracker/processor — **no** import of `pkg/regionproxy` in production ingestion code |

---

## CI and drills

| Suite | When | Blocks PR? |
| :--- | :--- | :--- |
| `mr_*` fault proofs in `scripts/test/run_resilience.sh` | `main` push | No (resilience job) |
| `bash scripts/test/mr_resilience_drill.sh` | Manual / [enterprise-resilience workflow](../.github/workflows/enterprise-resilience.yaml) | No |
| XDP compliance fingerprint | `scripts/ci/compliance.sh` | No XDP required for merge |

---

## Related docs

- [ARCHITECTURE.md](ARCHITECTURE.md) — appliance topology; Enterprise boxes marked optional
- [DEVELOPMENT.md](DEVELOPMENT.md) — daily dev; Enterprise drills linked, not in §8 main path
- [QUICKSTART.md](QUICKSTART.md) — single-VPS install only
