# Naming policy

Two layers: **product name** for non-technical readers vs **technical identity** in code, paths, and engineering docs. Legacy **`espx` / `eSPX` is removed** — not replaced by another internal brand.

**Status:** closed — [MILESTONES.md §11](MILESTONES.md#11-de-branding-espx--ad-event-processor--closed-2026-08-12). Open agent backlog: [`.cursor/MILESTONE.md`](../.cursor/MILESTONE.md).

---

## 1. Product name (public)

**BidShard** — used only where a non-engineer reads the text.

| Surface | Name | Examples |
| :--- | :--- | :--- |
| Customer / operator docs | **BidShard** | [QUICKSTART.md](QUICKSTART.md), [PILOT_LICENSE.md](PILOT_LICENSE.md) |
| Repository overview | **BidShard** | [README.md](../README.md) (product positioning, install UX) |
| Admin UI (default) | **BidShard** | `pkg/branding` → `BRAND_PRODUCT_NAME` default; web shell title |
| EULA / license UX | **BidShard** | `pkg/legal/EULA.txt`, license renewal copy in admin |
| Support / marketing URLs | configurable | `BRAND_SITE_URL`, `BRAND_SUPPORT_EMAIL` — no hardcoded vendor domain required in code |

**Not public (no BidShard in body text):** [DEVELOPMENT.md](DEVELOPMENT.md), [TRADEOFFS.md](TRADEOFFS.md), [BENCHMARKS.md](BENCHMARKS.md), [EDGE_CASES.md](EDGE_CASES.md), [PARSER_SECURITY.md](PARSER_SECURITY.md), [SHARDING_MILESTONE.md](SHARDING_MILESTONE.md), [MILESTONES.md](MILESTONES.md), [CUT_CANDIDATES.md](CUT_CANDIDATES.md), `.cursor/*` agent rules. Use **ad-event-processor** when referring to the software stack (no `aep` abbreviation).

[ARCHITECTURE.md](ARCHITECTURE.md): one-line product context may say BidShard; technical sections use component names (`tracker`, `control`, `processor`) and **ad-event-processor** for the stack.

---

## 2. Technical identity (internal)

Canonical technical name: **`ad-event-processor`** (kebab-case in paths, images, systemd; snake_case in env/metrics where required).

Database anchor (unchanged): **`ad_event_processor`** — Postgres `DB_NAME`, ClickHouse database, processor log line.

| Layer | Target | Deprecated (remove) |
| :--- | :--- | :--- |
| Go module (`go.mod`) | `github.com/<org>/ad-event-processor` | `module espx` |
| Import path prefix | `ad-event-processor/...` or module path | `espx/...` |
| CLI binary | `ad-event-processor` (`cmd/ad-event-processor`) | `cmd/espx`, `espx` binary |
| Installer script (ops) | `ad-event-processor-install.sh` | `bidshard-install.sh` optional symlink for docs only |
| Docker image | `ad-event-processor` | `bidshard` image tag, `ghcr.io/.../espx` |
| Host config dir | `/etc/ad-event-processor` | `/etc/espx` |
| Runtime sockets | `/run/ad-event-processor/redis/...`, `/run/ad-event-processor/postgresql` | `/run/espx/...` |
| Compose volume | `ad_event_processor_run` | `espx_run` |
| Env prefix | `AD_EVENT_PROCESSOR_*` | `ESPX_*` (read aliases one release) |
| Ingress schema | `ad_event_processor_native` | `espx_native` |
| Edge Prometheus | `ad_event_processor_edge_*` | `espx_edge_*` |
| BPF metrics / build tag | `ad_event_processor_bpf_*`, `ad_event_processor_bpf_trace` | `espx_bpf_*`, `espx_bpf_trace` |
| Go helpers | `ad_event_processor_go_build` | `espx_go_build` |
| Agent SLA rules file | `platform-sla.mdc` | `platform-sla.mdc` |
| Payment DB (optional) | `ad_event_processor_payment` | `espx_payment` |

**Keep:** Prometheus core prefix `ad_*` (already neutral).

**Forbidden in new code/docs (engineering):** `espx`, `eSPX`, `ESPX_`, `espx_`, `/run/espx`, `/etc/espx`.

---

## 3. UI and white-label

`pkg/branding` defaults:

- `BRAND_PRODUCT_NAME` default → **BidShard** (admin UI, alerts)
- Technical strings in logs/metrics → **ad-event-processor**, never espx or `aep`

Operators may override all `BRAND_*` env vars for white-label installs without forking code.

---

## 4. Migration (upgrades)

When Phase 3–5 land, operators with existing installs:

```bash
# Paths (example)
sudo mv /etc/espx /etc/ad-event-processor
# Compose: recreate stack after .env updates (AD_EVENT_PROCESSOR_* vars, REDIS_ADDRS socket paths)

# Env: set AD_EVENT_PROCESSOR_LICENSE_MODE=file; old ESPX_* still read for one release
```

Full checklist: [QUICKSTART.md](QUICKSTART.md#upgrading-an-existing-install) upgrade section and [NAMING.md §4](NAMING.md#4-migration-upgrades).

---

## 5. CI gates (planned)

| Script | Fails on |
| :--- | :--- |
| `scripts/ci/check_no_espx.sh` | `espx`, `ESPX_`, `espx_` outside allowlist (migration aliases, this doc) |
| Public doc lint (optional) | `espx` in QUICKSTART / PILOT_LICENSE / README |

Engineering docs must not introduce **espx**. Public docs must not introduce **espx**; they use **BidShard** for the product.
