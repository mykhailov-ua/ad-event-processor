# Advanced business logic and infra automation

Concepts to scale eSPX (Model 2/3) and reduce DevOps burden on self-hosted operators.

Operational acceptance criteria: this runbook and [DEVELOPMENT.md](../DEVELOPMENT.md) completed roadmap (P08, P13–P17, P26).

---

## 1. Zero-DevOps infrastructure (client-side)

Goal: run on a clean Linux VPS without a dedicated SRE.

### P08 — Installer doctor CLI (GAP-OPS-05)

| Probe | Checks |
| :--- | :--- |
| Kernel | XDP support via sysfs |
| sysctl | FD limit, TCP buffers, conntrack |
| Redis | RTT per shard; shard-0 reachability |
| ClickHouse | Write IOPS smoke |
| Disk | iogate latency budget |

Auto-tune on start: `GOMEMLIMIT`, `GOGC`, `PinnedWorkerPool` from RAM/CPU.

### P15 — Embedded ops dashboard (GAP-OPS-06)

| Feature | Detail |
| :--- | :--- |
| Metrics | Poll `/metrics` every 15 s |
| History | 24 h local TS (SQLite or CH) |
| Topology | Tracker / processor / Redis health map |
| Drift badge | From `ad_recon_drift_micro` |

### P26 — Redacted support debug bundle (GAP-SUP-01)

One-click `.tar.gz`: sanitized logs, pprof, version, license state (no key). No URLs, IPs, creatives.

---

## 2. Advanced business logic

### P13 — Campaign budget pacing (GAP-BIZ-01)

| Stage | Action |
| :--- | :--- |
| Cold | 7d hourly distribution from CH (or PG fallback) |
| Write | `pacing_ratio` in Redis per campaign |
| Hot | Snapshot read; probabilistic throttle (0 allocs) |

### P16 — Bid shading and floor optimizer (GAP-BIZ-02)

CH analysis: win rate vs floor → suggest or apply floor via management API. Dry-run default.

### P17 — Retargeting segments (GAP-BIZ-03)

Processor on conversion: add `user_id` hash to Redis Bloom/set. FilterEngine segment check on hot path.

### P07 — Margin guard and revenue share (GAP-BIZ-04)

Multi-leg ledger (`publisher_payout`, `operator_margin`). Auto-pause when RTB cost exceeds revenue threshold.

---

## 3. Operations (future)

| Feature | Gap |
| :--- | :--- |
| Signed binary hot-swap (`SO_REUSEPORT`) | TBD |
| Encrypted backup stream to S3 | TBD |
