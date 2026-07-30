# Advanced business logic and infra automation

Concepts to scale eSPX (Model 2/3) and reduce DevOps burden on self-hosted operators.

**GAP specs:** [.cursor/GAP_SPECS.md](../../.cursor/GAP_SPECS.md) — DoD, SLA, SQL, patterns, fault tests per gap.

---

## 1. Zero-DevOps infrastructure (client-side)

Goal: run on a clean Linux VPS without a dedicated SRE.

### GAP-OPS-05 — `espx doctor` and auto-tuning

| Probe | Checks |
| :--- | :--- |
| Kernel | XDP support via sysfs |
| sysctl | FD limit, TCP buffers, conntrack |
| Redis | RTT per shard; shard-0 reachability |
| ClickHouse | Write IOPS smoke |
| Disk | iogate latency budget |

Auto-tune on start: `GOMEMLIMIT`, `GOGC`, `PinnedWorkerPool` from RAM/CPU.

**Spec:** [GAP-OPS-05](../../.cursor/GAP_SPECS.md#gap-ops-05--zero-devops-espx-doctor)

### GAP-OPS-06 — Embedded lite dashboard

| Feature | Detail |
| :--- | :--- |
| Metrics | Poll `/metrics` every 15 s |
| History | 24 h local TS (SQLite or CH) |
| Topology | Tracker / processor / Redis health map |
| Drift badge | From `ad_recon_drift_micro` |

**Spec:** [GAP-OPS-06](../../.cursor/GAP_SPECS.md#gap-ops-06--embedded-lite-dashboard)

### GAP-SUP-01 — Redacted debug bundle

One-click `.tar.gz`: sanitized logs, pprof, version, license state (no key). No URLs, IPs, creatives.

**Spec:** [GAP-SUP-01](../../.cursor/GAP_SPECS.md#gap-sup-01--redacted-debug-bundle)

---

## 2. Advanced business logic

### GAP-BIZ-01 — Virtual Private Pacing (VPP)

| Stage | Action |
| :--- | :--- |
| Cold | 7d hourly distribution from CH (or PG fallback) |
| Write | `pacing_ratio` in Redis per campaign |
| Hot | Snapshot read; probabilistic throttle (0 allocs) |

**Spec:** [GAP-BIZ-01](../../.cursor/GAP_SPECS.md#gap-biz-01--smart-pacing-vpp)

### GAP-BIZ-02 — Bid shading / floor optimizer

CH analysis: win rate vs floor → suggest or apply floor via management API. Dry-run default.

**Spec:** [GAP-BIZ-02](../../.cursor/GAP_SPECS.md#gap-biz-02--bid-shading--floor-optimizer)

### GAP-BIZ-03 — Smart retargeting segments

Processor on conversion: add `user_id` hash to Redis Bloom/set. FilterEngine segment check on hot path.

**Spec:** [GAP-BIZ-03](../../.cursor/GAP_SPECS.md#gap-biz-03--smart-retargeting-segments)

### GAP-BIZ-04 — Margin guard & revenue share

Multi-leg ledger (`publisher_payout`, `operator_margin`). Auto-pause when RTB cost exceeds revenue threshold.

**Spec:** [GAP-BIZ-04](../../.cursor/GAP_SPECS.md#gap-biz-04--margin-guard--revenue-share)

---

## 3. Operations (future)

| Feature | Gap |
| :--- | :--- |
| Signed binary hot-swap (`SO_REUSEPORT`) | TBD |
| Encrypted backup stream to S3 | TBD |

---

## Backlog index

| ID | Task | Spec |
| :---: | :--- | :--- |
| GAP-OPS-05 | `espx doctor` + auto-tuning | [§](../../.cursor/GAP_SPECS.md#gap-ops-05--zero-devops-espx-doctor) |
| GAP-OPS-06 | Lite dashboard | [§](../../.cursor/GAP_SPECS.md#gap-ops-06--embedded-lite-dashboard) |
| GAP-BIZ-01 | VPP pacing | [§](../../.cursor/GAP_SPECS.md#gap-biz-01--smart-pacing-vpp) |
| GAP-BIZ-02 | Floor optimizer | [§](../../.cursor/GAP_SPECS.md#gap-biz-02--bid-shading--floor-optimizer) |
| GAP-BIZ-03 | Retargeting segments | [§](../../.cursor/GAP_SPECS.md#gap-biz-03--smart-retargeting-segments) |
| GAP-BIZ-04 | Margin guard | [§](../../.cursor/GAP_SPECS.md#gap-biz-04--margin-guard--revenue-share) |
| GAP-SUP-01 | Debug bundle | [§](../../.cursor/GAP_SPECS.md#gap-sup-01--redacted-debug-bundle) |
