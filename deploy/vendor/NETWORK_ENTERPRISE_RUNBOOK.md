# Network and Enterprise ops runbook

Internal. Sales, onboarding, and support for `network` and `enterprise` SKUs under manual crypto billing. Technical deploy references: [ENTERPRISE_DEPLOY.md](./ENTERPRISE_DEPLOY.md), [sku.yaml](./sku.yaml), [SALES.md](./SALES.md).

Customer-facing SLA text: **Appendix A** (Network) and **Appendix B** (Enterprise). Copy into Telegram or invoice attachment after buyer accepts [PUBLIC_OFFER.md](./PUBLIC_OFFER.md).

---

## 1. When to use which tier

| Buyer need | SKU | License highlights |
| :--- | :--- | :--- |
| 2-3 regions, slot migration, up to 10 hosts, <= 150k peak RPS | `network` | `multi_region`, `slot_migration`; no XDP, no platform campaign API |
| XDP edge, platform API sync, >150k RPS or >3 regions, fingerprint bind | `enterprise` | All Network features + `ebpf_xdp_edge`, `ad_platform_campaign_api`; `max_rps` uncapped in JWT (contractual cap in sizing sheet) |

**Downgrade path:** buyer on Scale with single region should stay on `scale` until they need `multi_region` or second production cell.

**Upsell triggers:**

- Second geographic ingest cell -> `network`
- NIC-level flood on tracker link, need SYN/PPS shed before nginx -> `enterprise`
- External ad platform campaign sync (Meta/Google bulk API worker) -> `enterprise`
- Declared sustained peak > 150k RPS -> `enterprise` with written RPS cap in sizing sheet

---

## 2. Pre-sales qualification (go / no-go)

Complete before sending invoice. Reject or defer to pilot if any hard fail.

| # | Question | Network minimum | Enterprise minimum | Fail |
| :--- | :--- | :--- | :--- | :--- |
| 1 | Offer accepted? | `trial_registry.json` row or vendor API | same | No paid JWT |
| 2 | Self-hosted VPS/metal under buyer control? | yes | yes | No SaaS promise |
| 3 | Declared peak RPS (sustained 5 min) | <= 150k | documented in sizing sheet | Over cap without enterprise + quote |
| 4 | Regions needed in first 90 days | 1-3 | 1-99 (document count) | >3 on network |
| 5 | Production hosts (tracker/redis/pg/ch) | <= 10 activations | <= 99 | Over activation cap |
| 6 | Linux kernel on edge (if XDP) | n/a | 6.1+ with BTF | Defer XDP or bare-metal upgrade |
| 7 | Ingress sees real client IP or CDN? | document | document + XDP limits if CDN | Mis-set expectations on XDP value |
| 8 | Payment | USDT (TRC20/ERC20 per invoice) | same | No fiat escrow in v1 |
| 9 | Operator skill | 1 Linux admin reachable in P1 | same + on-call for ingest | No warm body for P1 |
| 10 | Use case lawful / no CC Art. 361 ask | yes | yes | Revoke / no sale |

**Soft disqualifiers (pilot first):** no metrics stack, no staging host, cannot run `go run ./cmd/installer license host-id`, refuses topology diagram.

---

## 3. Discovery intake (mandatory before paid JWT)

Store answers in CRM note or `trial_registry` pending metadata. Re-run on major topology change.

### 3.1 Buyer profile

| Field | Example | Notes |
| :--- | :--- | :--- |
| `deployment_id` | UUID v7 | Same across pilot -> paid -> renewals |
| Legal label | "ACME media buying" | Display only; no EDRPOU required |
| Primary contact | Telegram @handle | Tier-1 channel |
| Technical contact | @ops_user | P1 bridge |
| Timezone | UTC+2 | Support window alignment |
| Traffic type | click + S2S postback | OpenRTB Y/N |
| Antifraud tier | rules + ML boost | Pilot often rules-only |

### 3.2 Expected load questionnaire

| Metric | Definition | Network template | Enterprise template |
| :--- | :--- | :--- | :--- |
| Peak ingest RPS | Max `/track` + `/click` sustained 5 min | _____ (cap 150k) | _____ (contract cap: _____) |
| Average RPS | Typical business hour mean | _____ | _____ |
| Daily events | Approximate | _____ | _____ |
| Burst factor | peak / average | <= 5x typical | document if higher |
| OpenRTB QPS | `/openrtb/bid` if enabled | _____ | _____ |
| Campaign count | active in PG | _____ | _____ |
| Postback fan-out | outbound HTTP from processor | _____ conn/s | _____ |
| Report/export jobs | concurrent heavy exports | <= 50 tenants | _____ |
| CH retention | months of raw events | _____ | _____ |
| Growth 6 mo | expected peak RPS | _____ | drives sizing headroom |

**Sizing rule:** provision for **1.5x declared peak RPS** on tracker pool aggregate. If buyer will not share load, assume pilot-tier sizing only and cap JWT at declared number.

### 3.3 Hardware audit checklist

Buyer fills; operator marks pass / risk / fail. Audit is **remote** (no datacenter visit in standard SKU).

#### Per role (minimum production layout)

| Role | Count (min) | vCPU | RAM | Disk | NIC | Network pass criteria |
| :--- | ---: | ---: | ---: | :--- | :--- | :--- |
| Tracker + nginx edge | 2+ | 16 each | 32 GiB | 100 GiB NVMe | 10 GbE preferred | IRQ spread, `rps_cpus` documented |
| Redis state (4 shards) | 4 | 8 | 32 GiB | 200 GiB NVMe | 10 GbE | `noeviction`, latency < 1 ms LAN |
| Redis optional replica | 0-4 | 8 | 32 GiB | same | same | For quorum multi-region |
| Postgres (global) | 1 (+ replica rec.) | 8 | 16 GiB | 500 GiB SSD | 1 GbE+ | PITR backup exists |
| ClickHouse | 1+ | 16 | 64 GiB | 1 TiB+ NVMe | 10 GbE | Separate from tracker |
| Control + admin | 1 | 4 | 8 GiB | 50 GiB | 1 GbE | Not on tracker host |
| Processor + workers | 1-2 | 8 | 16 GiB | 200 GiB | 1 GbE | Broker WAL disk headroom |
| region-proxy (per cell) | 1 | 4 | 8 GiB | 100 GiB | 1 GbE | WAL disk not on tmpfs |
| edge-xdp host (enterprise) | 1 per ingress NIC | 8 | 16 GiB | 50 GiB | **dedicated ingress** | BTF, `CAP_BPF`, not behind L4 NAT |

#### Host audit commands (buyer runs, paste output)

```bash
uname -r                                    # kernel >= 6.1 for XDP
test -r /sys/kernel/btf/vmlinux && echo btf_ok
nproc
free -h
lsblk -o NAME,SIZE,TYPE,MOUNTPOINT
ip -br link
ethtool -l eth0 2>/dev/null | head -5     # NIC queues
go run ./cmd/installer license host-id      # HWID for JWT
```

#### Audit outcomes

| Result | Meaning | Action |
| :--- | :--- | :--- |
| **Pass** | Meets minimums for declared peak | Proceed to JWT |
| **Risk** | Undersized <= 20% or single-AZ | JWT + written resize deadline in SLA |
| **Fail** | < 1.5x headroom, SMR disk on Redis, no backup | No production JWT until remediated |

Deliverable: **Hardware audit memo** (1-2 pages): table above filled, pass/risk/fail, resize list. Network: async within **5 business days**. Enterprise: within **3 business days** + NIC IRQ plan if XDP.

### 3.4 Network topology questionnaire

| # | Topic | Capture |
| :--- | :--- | :--- |
| 1 | Diagram | ASCII or PNG: DNS -> CDN? -> LB? -> nginx -> tracker -> Redis/PG/CH |
| 2 | TLS termination | nginx on tracker host / CDN / separate LB |
| 3 | Client IP visibility | `X-Real-IP` / `X-Forwarded-For` trust chain |
| 4 | Egress | Postbacks to affiliate networks; SMTP if alerts |
| 5 | Regions | Region codes, which cell is global (`AD_EVENT_PROCESSOR_REGION_CODE=0`) |
| 6 | Uplink | Regional -> global `:8188` path, API key rotation |
| 7 | Firewall | Who can reach admin `:8188`, Redis, Postgres |
| 8 | Monitoring | Prometheus/Grafana? buyer provides scrape endpoint for remote review |
| 9 | Backup | PG PITR, CH backup, Redis RDB/AOF policy |
| 10 | Incident bridge | Telegram group invite for P1 |

**Multi-region minimum topology (network):**

```
[Global region code 0]
  control, Postgres primary, Redis shard 0 (quorum), optional global CH

[Regional cell 1..N]
  tracker pool, processor (MULTI_REGION_ENABLED=1), region-proxy
  local Redis for ingest path
  uplink HTTPS -> global /api/v1/region/ingest/batch
```

**Enterprise add-on:** XDP on **dedicated ingress NIC** on tracker edge host(s); parallel L7 nginx blacklist remains. CDN-terminated TCP: document that XDP sees edge IPs only.

Deliverable: **Topology review memo** with risks (CDN+XDP, single-AZ global PG, uplink without TLS). Network: **1x 60 min** video call. Enterprise: **2x 90 min** workshops + written sign-off.

---

## 4. Commercial flow (crypto)

1. Offer acceptance on file (`accept-offer` / bot `/accept` / vendor API).
2. Complete sections 3.2-3.4 (load + audit + topology).
3. Send [INVOICE.md](./INVOICE.md): SKU, USDT amount, network (TRC20/ERC20), `deployment_id`, month count.
4. Buyer pays; operator records tx hash + amount + date in CRM.
5. Issue JWT:

```bash
go run ./cmd/license-issue --sku network \
  --customer "Buyer label" \
  --deployment-id "<uuid>" \
  --hwid-v2 "<hash>" \
  --telegram-id "<id>" \
  --mark-converted
```

Enterprise with contractual RPS cap (optional env at issue):

```bash
go run ./cmd/license-issue --sku enterprise \
  --customer "Buyer label" \
  --deployment-id "<uuid>" \
  --hwid-v2 "<hash>"
```

6. Attach **Appendix A or B SLA** PDF/text in Telegram.
7. Schedule onboarding kickoff (within SLA onboarding window).

Renewal: same `deployment_id`, new JWT before `exp` + 7d grace. Support hours do not reset mid-month on renewal.

---

## 5. Network tier onboarding runbook

### Phase 0 — Kickoff (day 0-1)

| Step | Owner | Done when |
| :--- | :--- | :--- |
| Create shared Telegram group | Sales | Buyer ops invited |
| Send installer URL | Sales | `https://bidshard.com/releases/ad-event-processor-installer.tar.gz` |
| Confirm `deployment_id` | Ops | Matches registry |
| Deliver paid JWT | Ops | Buyer applied in admin |
| Share ENTERPRISE_DEPLOY Part 2 only | Ops | Multi-region, not XDP |

### Phase 1 — Staging cell (day 1-5)

| Step | Command / check |
| :--- | :--- |
| Install global region | `install.sh` or compose `full` profile on staging |
| License apply | Admin Settings or `license-apply` |
| Smoke ingest | `curl` `/track` 202, CH row or stream metric |
| Enable multi-region JWT features | `multi_region`, `slot_migration` true in effective entitlements |
| Stand regional cell | `MULTI_REGION_ENABLED=1`, `REGION_CODE=1`, region-proxy up |
| Uplink test | `region-proxy` -> global batch 200 |
| Resilience read | Buyer runs `multi_region_resilience_drill.sh` on staging (optional assisted) |

### Phase 2 — Production cutover (day 5-14)

| Step | Detail |
| :--- | :--- |
| DNS / traffic switch | Low-TTL cutover; keep pilot host read-only 24h |
| Slot migration | If Redis shard move planned, use admin migration job; dual-write window |
| Monitor | `ad_http_request_duration_seconds` p95 < 50 ms, p99 < 80 ms on control cohort |
| Budget invariant | `AssertBudgetInvariant` after first spend hour |
| Sign-off | Buyer confirms peak within 150k RPS license |

### Phase 3 — Steady state (month 1+)

| Cadence | Activity |
| :--- | :--- |
| Weekly | Review Prometheus screenshots (buyer-provided) if no remote access |
| Monthly | 30 min health call (included in engineer hours) |
| Quarterly | Re-validate load questionnaire if traffic +30% |

### Phase 4 — Escalation triggers

| Signal | Action |
| :--- | :--- |
| p99 > 80 ms 30 min | P2 sizing review |
| `EXPIRED` ingest block | P1 license renewal |
| Regional uplink lag | P2 region-proxy / firewall |
| Redis OOM / evictions | P1 capacity |

---

## 6. Enterprise tier onboarding runbook

All Network phases **plus**:

### Phase E1 — XDP readiness (before attach)

| Check | Detail |
| :--- | :--- |
| Kernel BTF | `/sys/kernel/btf/vmlinux` readable |
| NIC | `INGRESS_INTERFACE` dedicated; not `lo` in prod |
| Capabilities | `privileged` container or bare `edge-xdp` with `CAP_BPF`, `CAP_NET_ADMIN` |
| License | `ebpf_xdp_edge: true` effective |
| CDN disclaimer | Signed in topology memo if CDN used |

Deploy reference: [ENTERPRISE_DEPLOY.md](./ENTERPRISE_DEPLOY.md) Part 1.

```bash
make gen bpf-dev
docker compose -f deploy/compose/docker-compose.yaml --profile enterprise-xdp up -d edge-xdp
```

Verification:

```bash
go test ./internal/edge/ -short -run TestTokenBucket_ -count=1
curl -s localhost:9191/metrics | head
```

### Phase E2 — Platform campaign API (if used)

| Step | Detail |
| :--- | :--- |
| Feature gate | `ad_platform_campaign_api: true` |
| Control env | `CONTROL_ENABLE_PLATFORM_CAMPAIGN_SYNC=1` |
| Credentials | Buyer supplies platform tokens; store in buyer PG only |
| Outbox | Confirm worker processes without blocking `/track` |

### Phase E3 — Enterprise cutover

| Step | Detail |
| :--- | :--- |
| XDP shadow | Attach in log-only / metrics-only if available; then enforce drop |
| Load test | Buyer traffic or agreed window; abort if control p99 > 80 ms 30 s |
| RPS cap | Document contractual cap even if JWT uncapped |
| Runbook handoff | Buyer on-call sheet + escalation phone/Telegram |

---

## 7. Support operations

### Severity definitions

| Sev | Definition | Examples |
| :--- | :--- | :--- |
| **P1** | Production ingest blocked or spend invariant broken | `EXPIRED`, all trackers 5xx, budget drift |
| **P2** | Degraded: elevated latency, one region down, XDP mis-sync | p99 > 80 ms, uplink retry storm |
| **P3** | Non-urgent: admin UI, reports lag, how-to | Export slow, RBAC question |
| **P4** | Feature request / roadmap | New integration |

### Tier-1 (first responder)

Tier-1 = operator on `@bidshardsupportbot` / `support@getbidshard.com`:

- Acknowledge ticket, collect `deployment_id`, severity, logs (last 200 lines), recent change
- Run playbook: license state, Redis ping, compose ps, `curl /health`
- Escalate to engineering within SLA response window if not resolved in 30 min (P1) or 2 h (P2)

Engineering = same team; "Tier-1" is the **contractual first response**, not a separate NOC vendor.

### Remote access policy

Default: **no standing SSH**. Buyer pastes logs/metrics. Enterprise may grant temporary VPN/jump box **read-only** by written Telegram approval (time-boxed 48h).

### Engineer hours (included)

Consumed by: onboarding calls, audit memo, topology review, assisted debug, upgrade guidance. **Not** custom feature dev (quote separately).

| Month | Network included | Enterprise included |
| :--- | ---: | ---: |
| Month 1 (onboarding) | 8 h | 20 h |
| Month 2+ | 4 h / month | 12 h / month |
| Overage | USD 120/h USDT, pre-approve | USD 150/h USDT, pre-approve |

Track hours in CRM; warn buyer at 80% consumption.

---

## 8. Product performance targets (engineering)

Referenced in PUBLIC_OFFER section 10.3 as development targets. SLA appendices tie **buyer sizing compliance** to these targets.

| Surface | Metric | Target (sized deployment) |
| :--- | :--- | :--- |
| Tracker ingest | `ad_http_request_duration_seconds` p95 | < 50 ms |
| Tracker ingest | p99 | < 80 ms (hard ceiling 100 ms) |
| Unified-filter Lua | p99 per shard | < 10 ms |
| Multi-region uplink | Regional WAL drain after partition | < 120 s RTO (operator drill) |
| Budget | Postgres `current_spend` vs Redis | `AssertBudgetInvariant` +/- 1 micro-unit |

If buyer declines audit minimums, targets are **best-effort** only.

---

## 9. Evidence pack (per paid customer)

| Artifact | Storage |
| :--- | :--- |
| offer_version + accepted_at | `trial_registry.json` |
| Invoice + tx hash | CRM |
| Load questionnaire | CRM / ticket |
| Hardware audit memo | CRM |
| Topology review memo | CRM |
| SLA appendix sent (A or B) | CRM timestamp |
| JWT `deployment_id`, `exp` | issue log |
| Onboarding hours log | CRM |

---

# Appendix A — Customer SLA summary (`network`)

**SKU:** `network` — USD 1,399 / month (USDT). **Support window:** 10:00-22:00 Europe/Kyiv, Monday-Saturday. **Channels:** Telegram (@bidshardsupportbot) primary; support@getbidshard.com secondary.

### Included services

| Item | Network tier |
| :--- | :--- |
| Tier-1 support | Yes — first response by operator on-call roster |
| Onboarding engineer hours | 8 h month 1, then 4 h / month |
| Hardware audit (remote) | Checklist review + memo within 5 business days of complete intake |
| Topology review | One 60 min session + written risks |
| Load fit review | Declared peak vs 150k RPS license cap |
| License issuance / renewal | Yes |
| Defect triage (repro on reference compose) | Yes |

### Response targets (commercial courtesies)

| Severity | First response | Update cadence | Restoration target |
| :--- | :--- | :--- | :--- |
| P1 | 4 h within support window | every 2 h | best-effort same day; depends on buyer access |
| P2 | 1 business day | daily | best-effort 3 business days |
| P3 | 3 business days | weekly | next release or workaround |
| P4 | 5 business days | n/a | roadmap |

**Outside window:** P1 ack within next window open; no penalty credit in standard SKU.

### Buyer responsibilities

- Maintain topology and contacts current
- Provide metrics/logs within 4 h of P1 request
- Stay within license: 150k peak RPS, 10 hosts, 3 regions
- Run backups and OS security on own infrastructure
- Accept crypto payment terms (final, no chargeback)

### Excluded

24/7 outsourced NOC, on-site visits, managed hosting, custom development, traffic forensics, legal/GDPR consulting, CDN contract negotiation.

### Ingest performance (when audit = pass)

If deployment matches approved hardware audit and declared load, Licensor targets tracker p95 < 50 ms and p99 < 80 ms on buyer's control cohort during normal operation. Breach due to buyer undersizing, DDoS beyond SYN/PPS limits, or third-party network is out of scope.

---

# Appendix B — Customer SLA summary (`enterprise`)

**SKU:** `enterprise` — from USD 2,999 / month (USDT); custom RPS cap in sizing sheet. **Support window:** 09:00-24:00 Europe/Kyiv, seven days. **Channels:** dedicated Telegram thread + support@getbidshard.com.

### Included services

| Item | Enterprise tier |
| :--- | :--- |
| Tier-1 support | Yes — named primary + backup contact in welcome packet |
| Onboarding engineer hours | 20 h month 1, then 12 h / month |
| Hardware audit (remote) | Full memo within 3 business days; NIC/XDP IRQ plan |
| Topology review | Two 90 min workshops + written sign-off |
| Load fit review | Contractual peak RPS documented; JWT may be uncapped technically |
| XDP attach assistance | Guided deploy + rollback drill |
| Multi-region cutover assistance | Uplink + quorum checklist |
| Platform API sync setup | One integration session if feature enabled |
| Priority defect queue | Yes |

### Response targets (commercial courtesies)

| Severity | First response | Update cadence | Restoration target |
| :--- | :--- | :--- | :--- |
| P1 | 1 h within support window | every 1 h | best-effort 8 h |
| P2 | 4 h | every 4 h | best-effort 2 business days |
| P3 | 1 business day | twice weekly | next patch or workaround |
| P4 | 3 business days | n/a | roadmap |

**P1 outside window:** ack within 2 h if on-call phone/Telegram ping enabled by buyer.

### Buyer responsibilities

- Same as Network, plus: maintain XDP-capable kernel, document CDN/LB constraints, approve change windows for edge attach
- Enterprise fingerprint bind: notify before hardware migration (new HWID)

### Excluded

Same as Network, plus: writing buyer's platform API OAuth apps, kernel patches, NIC firmware, colo smart-hands (buyer contracts smart-hands; we provide runbook).

### Ingest and edge performance (when audit = pass)

Tracker targets as Network. XDP layer: local drop path for listed floods; not a guarantee against all DDoS. Enterprise buyer receives annual resilience checklist (`multi_region_resilience_drill.sh`, XDP rollback doc).

---

## Related docs

| Doc | Use |
| :--- | :--- |
| [VENDOR_OPS_RUNBOOK.md](./VENDOR_OPS_RUNBOOK.md) | Pilot + paid CLI |
| [ENTERPRISE_DEPLOY.md](./ENTERPRISE_DEPLOY.md) | XDP + multi-region technical |
| [INVOICE.md](./INVOICE.md) | USDT invoice template |
| [PUBLIC_OFFER.md](./PUBLIC_OFFER.md) | Binding license terms |
