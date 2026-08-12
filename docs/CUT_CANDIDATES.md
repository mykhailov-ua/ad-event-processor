# Deprecation and Freeze Candidates (Appliance SKU)

Inventory of system components and capabilities that introduce operational or resume-driven overhead relative to the primary commercial product: **self-hosted single-VPS** (for ad networks, media buyers, and publishers).

Canonical deployment scenario: `deploy/compose` using profile `single_vps` + installer + `docs/QUICKSTART.md`.

Inclusion criteria:
- Does not increase average deal size for the boxed product (appliance SKU).
- Creates maintenance, installation, and documentation costs without direct revenue generation.
- Contradicts actual shipping artifacts (marketing noise / abandoned manifests).

Classification categories:
- **CUT** - Completely remove from codebase, documentation, or navigation.
- **FREEZE** - Retain in git history for enterprise customers (SOW / Enterprise); exclude from installer, sales pitch, and QUICKSTART.
- **KEEP** - Core functionality; protects revenue generation or the boxed distribution package.

This document is intended strictly for engineering personnel.

Related documents: [QUICKSTART.md](QUICKSTART.md), [ARCHITECTURE.md](ARCHITECTURE.md), [TRADEOFFS.md](TRADEOFFS.md), `.cursor/INSTALLER.md`.

---

## 1. Scheduled for Removal (CUT)

| Candidate | Evidence in Code / Docs | Reason for Removal | Effort |
| :--- | :--- | :--- | :--- |
| Claims of Privacy Sandbox / Topics / PAAPI support | `README.md`, `ARCHITECTURE.md` Section 10 (no Topics/PAAPI implementations exist; consent compliance is not Sandbox) | Inaccurate marketing claims; undermines product credibility | S |
| Non-existent "CTV live / concurrent streams" support | Discrepancies between `README.md` and RTB/CTV checklists | Declarations without functional code implementations | S |
| Kubernetes as standard deployment option | ~~`deploy/k8s/**`~~ **CUT (2026-08):** manifests removed; k3s scripts archived under `deploy/enterprise/archive/k8s/`; installer profile `k8s_k3s` removed | Product ships via Compose `single_vps` only | M |
| Terraform / k3s infrastructure manifests | ~~`deploy/terraform/**`~~ removed; k3s ops scripts in `deploy/enterprise/archive/k8s/` (not in installer/CI) | Unnecessary orchestrator; out of scope for single-VPS appliance | M |
| Abandoned environment files (orphan env stubs) | `deploy/management/`, `deploy/payment/` | Artifacts of pre-monolith architecture | S |
| Prometheus targets for deleted microservices | `deploy/monitoring/prometheus.yaml` (targets `auth`, `payment`, `management` despite single `control` binary) | Empty metrics, misleading monitoring visibility | S |
| `tracker-quic` binary | `cmd/tracker-quic` | TLS termination handled by Caddy/Nginx; redundant binary | S |
| BPF / purgatory documentation in client docs | `docs/BENCHMARKS.md`, `cmd/bpf-collector`, `deploy/dev/bpf/` | Internal developer debugging tools, not part of appliance SKU | S |
| Multi-region scenario as mandatory dev path | `docs/DEVELOPMENT.md` Section 8, `scripts/fault/mr_resilience_drill.sh`, `deploy/multi-region/` | Excessive process overhead for single-VPS buyers | S-M |
| Excessive Docker Compose profiles in primary flow | Profiles `analytics-ml`, `multi-region`, `network-operator`, `infra` alongside appliance | Blurs the single-click installation proposition | S |
| Rendered configuration artifact stubs | `*.rendered.yaml` and similar files | Repository clutter | S |
| Incorrect XDP path in documentation | Code generation map in `DEVELOPMENT.md` references incorrect `deploy/edge-xdp` instead of `deploy/edge/xdp/` | Initial build failure when following documentation | S |
| Outdated installer blockers | `.cursor/INSTALLER.md` marks G1/G2 as blockers despite complete tiers | Forces agents to re-fix obsolete issues | S |
| Broken references to deleted `scripts/deploy/*` | Invocation calls from `scripts/ops/` and tests to non-existent directories | Deployment errors during script execution | S |

---

## 2. Feature Freeze (FREEZE - Optional / Enterprise SOW)

| Candidate | Evidence | Governance Rule |
| :--- | :--- | :--- |
| Multi-region proxying (`region-proxy`) | `cmd/region-proxy`, `pkg/regionproxy/`, `deploy/broker/` | Exclude from base distribution; enable strictly under explicit Enterprise contracts |
| `fraud-scorer` / `ivt-detector` / `deploy/ml` | `cmd/fraud-scorer`, `cmd/ivt-detector`, `cmd/ml-*`, profile `analytics-ml` | Move to optional profile; hot path relies on blocklists and boost snapshots |
| `edge-xdp` / `edge-bpf-sync` | `cmd/edge-xdp`, `cmd/edge-bpf-sync`, `deploy/edge/` | Omitted from `single_vps`; primary filtering handled by Nginx Lua |
| Elastic sharding / `campaign-shard` / migrations | Environment variables `ELASTIC_SHARDING_*`, `cmd/campaign-shard`, `DEVELOPMENT.md` Section 9 | Cluster feature; appliance relies on fixed Redis masters |
| DFA parsers for HTTP/2 and HTTP/3 in tracker | `handler_http2.go`, `http3_frame*.go` | Pause feature expansion; Edge proxies HTTP/1.1 traffic to tracker |
| Per-node Grafana / Alertmanager / Telegram stack | `deploy/monitoring/**`, `cmd/alertmanager-telegram` | Move to `--profile monitoring`; restrict metric collection to tracker, control, processor |
| Log collectors and rotators (log-shipper / compactor / evacuator) | `cmd/log-*` | Node utility scripts; excluded from base SKU |
| OpenRTB Exchange enabled by default | `internal/rtb/`, `internal/openrtb/`, `RTB_PRODUCTION_RUNBOOK.md` | Preserve code; default `RTB_MODE=off`; do not lead sales pitches with SSP functionality |
| Strict SPO features (schain / `sellers.json`) | Parsing schain and exporting supply in OpenRTB; bullet in README | Freeze development, lower priority in marketing materials |
| Frontend backlog from FRONTEND Section 16 | `.cursor/FRONTEND.md` | Internal engineering tasks; prioritize customer P0 requests |
| Pilot build Dockerfiles | `Dockerfile.pilot`, `Dockerfile.pilot-ingest` | Restrict usage to pilot delivery pipeline |
| Legacy naming (removed) | was `go.mod` `espx`, `ESPX_*`, `/run/espx` | **Done** — internal **ad-event-processor**; public docs keep **BidShard** — [NAMING.md](NAMING.md) |
| Redis HA / Sentinel setups (6 Redis instances) | Profile `infra`, Sentinel overlays; documentation discrepancies (4 vs 6 nodes) | Standardize Redis instance counts for appliance; supply HA via Enterprise SOW |
| Process separation for network-operator / payment NodePort | Profile in DEVELOPMENT; Kubernetes service `service-payment*` | Payment processing executes within `control` binary |
| Optional CI workflows (terraform-validate, fraudtrain) | `.github/workflows/*` | Trigger via path filters; do not block critical code merges |

---

## 3. Retain Without Modification (KEEP)

| Component | Operational Purpose |
| :--- | :--- |
| `pkg/broker` engine and `cmd/broker` (Tiered Event Bus) | Memory-mapped disk WAL for offloading Redis RAM, guaranteeing event durability during crashes (Zero-Loss Crash Recovery), and supporting log replay |
| Tracker Lua filters / FilterEngine / Redis budgets | Prevents campaign budget overspending |
| Outbox + processor + PostgreSQL ledger (`balance_ledger`) | Precise financial settlement and reconciliation |
| Docker Compose `single_vps` + installer + embedded admin UI | Core product offering (appliance SKU) |
| Nginx Lua sharding matching `StaticSlotSharder` | Correct traffic routing to tracker instances |
| Handlers `POST /track`, `GET /click`, postback endpoints | Critical traffic ingestion and monetization path |
| Core ClickHouse reporting | Essential analytics and customer retention features |
| Privacy consent tables (GDPR / Consent) without Sandbox branding | Regulatory legal compliance |
| Offline pilot licensing | On-premises commercial sales model |
| StaticSlot + CRC32 routing topology | Core load distribution model |

---

## 4. Naming Technical Debt

| Historical Name | Current System State | Required Remediation Action |
| :--- | :--- | :--- |
| Service `management` / `MANAGEMENT_PORT` / Prom job | Binary executable `control` | Rename or remove alongside Kubernetes/monitoring cleanup |
| Container / release image | Tags `bidshard`, `ghcr.io/.../bidshard` | Standardize on **`ad-event-processor`** (matches DB + processor role) |
| Product (public) | `README`, `QUICKSTART`, `PILOT_LICENSE`, admin UI | Keep **BidShard** for non-tech readers — [NAMING.md](NAMING.md) |
| Go module + paths | was `module espx`, `/etc/espx`, `ESPX_*` | **Done** — **ad-event-processor** |
| Redis documentation configuration (4 vs 6 nodes) | Discrepancies with Docker Compose | Align Compose, `QUICKSTART.md`, and `INSTALLER.md` |

---

## 5. Recommended Execution Order

1. **Marketing and Claims Cleanup:** Remove Privacy Sandbox and unimplemented CTV references from `README.md` and `ARCHITECTURE.md`.
2. **Deployment Footprint:** Standardize documentation on self-hosted Compose + installer. Mark Kubernetes/Terraform as unsupported in the appliance SKU (archive or remove broken manifests).
3. **Appliance Optimization:** Disable heavy monitoring profiles in `single_vps`; standardize Redis node count; remove unused environment directory stubs.
4. **Script and Documentation Scrub:** Fix `scripts/deploy` calls; resolve `INSTALLER.md` inconsistencies; remove multi-region setups from the base development guide.
5. **Documentation Hierarchy:** Maintain customer-facing `QUICKSTART.md`, streamlined `ARCHITECTURE.md`, and `PILOT_LICENSE.md`. Reclassify `BENCHMARKS.md`, `EDGE_CASES.md`, `TRADEOFFS.md`, and this file as internal engineering documentation.
6. **Auxiliary Binaries:** Retain frozen binaries in the repository, but exclude them from commercial pitches and default profiles.
7. **Naming:** **Done** — public **BidShard**, internal **ad-event-processor** — [NAMING.md](NAMING.md), [MILESTONES.md §11](MILESTONES.md#11-de-branding-espx--ad-event-processor--closed-2026-08-12).

---

## 6. Audit Criteria (Resume-Driven Overhead)

A component or feature must be scheduled for removal (CUT) or freeze (FREEZE) if two or more of the following conditions are met:

- [ ] Requires an auxiliary orchestrator (Kubernetes, Terraform, second event bus).
- [ ] Omitted from the `single_vps` profile and not required in the tracking -> budget -> invoice path.
- [ ] Described in `README.md` with greater ambition than implemented in the installer.
- [ ] Contains broken code or references deleted microservices.
- [ ] Is never raised by potential customers during initial product demonstrations.
- [ ] Increases server RAM/CPU consumption without increasing license contract value.

---

## 7. Core Operating Policy

Sell **tracking -> budget -> financial settlement -> administration on a single server**.  
Any capability promoting **clustering, multi-region setups, ML/XDP/QUIC sidecars, Privacy Sandbox, or infrastructure complexity** must be removed or frozen until explicitly funded by an Enterprise customer contract.
