# BidShard Antifraud Overview

Internal document for vendor sales, presales, and operators. Technical details can be found in [ARCHITECTURE.md](../../docs/ARCHITECTURE.md) and `.cursor/rules/edge.mdc`.

---

## How It Works

BidShard’s antifraud architecture is divided into the **hot path** (executing on every click/impression with ultra-low latency, p99 < 80 ms) and the **cold path** (asynchronous batch analytics, machine learning, and automatic IP quarantine syncing).

| Processing Layer | Execution Scope | Latency Impact |
| :--- | :--- | :--- |
| **Perimeter (Edge / XDP)** | Nginx OpenResty Lua, optional eBPF kernel drops | Microseconds (0–1 ms) |
| **Hot Ingestion Signals** | Tracker `FilterEngine` in-memory checks | Nanoseconds to milliseconds (0–5 ms) |
| **IVT Analysis & ML** | `ivt-detector` rules, `fraud-scorer` LightGBM | 5–15 minutes (asynchronous batch) |
| **Operator Control** | Admin UI and Web/REST API | On-demand |

*Hot-Path Performance Guarantee:* Heavy machine learning models (LightGBM/ONNX) are **never** evaluated synchronously inside the redirect/ingestion loop. Instead, they run in background batch processes and publish risk boosts (`ml:score:boost:{campaign_id}`) directly to Redis. The hot-path tracker simply reads these pre-computed boosts in-memory in under 90 nanoseconds.

---

## The Three Reaction Levels

When an event triggers the antifraud engine, BidShard reacts based on the severity of the threat:

| Level | Trigger Condition | System Behavior |
| :--- | :--- | :--- |
| **L1 - Reject (Block)** | ≥ 2 strong signals OR matching L3 blacklists | `/track` returns a fake **HTTP 202 (Accepted)** without debiting budget or firing postbacks. `/click` returns **HTTP 204 (No Content)** or routes to a safe-view page. |
| **L2 - Shadow** | 1 strong signal, weak signals, or suspect-block | The event is fully accepted but flagged as `shadow_event` in ClickHouse. Visible in analytical dashboards without false-blocking real users. |
| **L3 - Blacklist** | IP address present in `blacklist:fraud` | Hard block. The edge/nginx layer can reject the request with a **HTTP 403** before it even hits the tracker. |

### Ghost IVT (Phantom Bypass)
When `ghost_ivt_enabled` is active on a campaign, BidShard uses a sophisticated defense mechanism: it fake-accepts suspicious traffic with standard 202/204 codes. The bot or spy tool receives a successful crawl response, but no campaign budget is debited, and no Conversions API (CAPI) or postback webhooks are sent out. This prevents competitor spy tools from knowing they have been cloaked.

---

## Hot-Path Signals & Network Auditing

Signals are evaluated in real time on the tracker, building up a cumulative `fraud_score` (0–100) and recording a `fraud_reason` parameter:

| Code | Signal | Base Weight | Severity | Description |
| :--- | :--- | :---: | :--- | :--- |
| `datacenter_ip` | Datacenter / Server IP (GeoIP) | 45 | L1 | Intercepts cloud hosting, Tor nodes, and public scrapers. |
| `low_ttc` | Low Time-To-Click | 45 | L1 | Detects clicks arriving faster than human capability after an impression. |
| `tls_blocklist` | TLS Handshake Fingerprint (JA3/JA4) | 45 | L1 | Identifies known bot frameworks and scrapers at the SSL/TLS layer. |
| `l3_blocklist` | Hard Blacklisted IP | 100 | L3 | Instantly blocks traffic from active, verified fraud IPs. |
| `missing_imp_ts` | Missing Impression Timestamp | 35 | L2 | Catches click-hijacking and click-injection scripts. |
| `device_mismatch` | User-Agent vs. Sec-CH-UA Headers | 35 | L2 | Detects browser impersonation (e.g., spoofing Chrome UA on non-Chrome engines). |
| `tcp_mss_anomaly` | Anomalous TCP MSS Header | 35 | L2 | Exposes virtualized OS environments via passive network analysis (`X-TCP-MSS`). |
| `os_fingerprint_mismatch` | OS Network Fingerprint vs. UA Family | 35 | L2 | Checks if initial TCP TTL matches the declared User-Agent family (`p0f-lite` rules). |
| `ipv4_rotation` | Sticky Subnet Rotation Velocity | 35 | L2 | Detects automated IP rotation farms within matching `/24` subnets. |
| `residential_proxy` | Residential Proxy IP Signature | 35 | L2 | Matches known proxy farm pools in the hot cache. |
| `attestation_missing` | Missing JS Attestation Token | 35 | L2 | Catches bots that fail to execute interactive Javascript stubs on `/click`. |

*Threshold Tuning:* Operators can set custom campaign tier thresholds: **pass -> suspect -> ivt -> block** (defaults: 30 / 60 / 80 / 100). Ready-made presets include: `conservative`, `balanced`, `aggressive`, `gray_market`, and `social_in_app`.

---

## Specialized Antifraud Presets

### 1. Social In-App (Optimized for Facebook, TikTok, Instagram WebViews)
In-app mobile browsers (WebViews) within major social apps have unique, often messy TLS and network signatures that can trigger false-positive alarms on generic trackers.
- **Mobile-Carrier Filtering:** Restricts traffic strictly to mobile carriers (`mobile_only`), allowing legitimate mobile web users while blocking server-side scraping, VPNs, and datacenter proxy nodes.
- **TLS Safe-Pass on `/click`:** When a visitor arrives via known social app markers (`FBAN`, `FBAV`, `musical_ly`, `Instagram`), standard strict TLS blocks are bypassed to prevent blocking legitimate buyers. JS attestation remains fully active in the background.

### 2. Gray Market (Maximum Landing Protection & Cloaking)
The ultimate shield for aggressive or highly targeted media-buying campaigns.
- **JS Attestation & Safe Page:** Enables interactive browser verification combined with Safe Page (cloaking) routing.
- **Silent Human Auditing:** When the visitor lands on a landing page, a lightweight, non-intrusive JS probe tests the browser for WebGL/canvas rendering capabilities, WebRTC audio support, real touch events, and mouse movement curves (Bezier curves), silently weeding out advanced automated bot networks.
- **Link Signing:** Restricts landing pages to signed links with strict time-to-live (TTL) limits, rendering shared or leaked links useless.

---

## High-Performance Ingestion Filters (Go Precheck)

To maintain an ultra-fast, zero-allocation hot path, BidShard extracts deterministic checks out of Lua scripts and executes them locally in the Go `FilterEngine` **before** executing heavy Redis operations:

| Filter Gate | Location | Lua Elimination Impact |
| :--- | :--- | :--- |
| **Placement Blacklist** | `UnifiedFilter.Check` (Go in-memory cache) | Eliminated `HEXISTS` calls from Redis scripts |
| **Ingress RPS/RPD** | `EntitlementsFilter` (Go local atomic increment) | Eliminated `max_rpd` tracking from Redis scripts |
| **Fraud Blacklist** | `UnifiedFilter.Check` (Go in-memory map) | Eliminated `SISMEMBER` checks from Redis scripts |
| **Time-To-Click (TTC)** | `applyGoTTC` (Go memory) | Evaluates low-TTC signals without executing Lua |

By executing these checks locally on the Go side, the Redis shard scripts are kept slim and focused entirely on atomic spend debiting and final click deduplication.

---

## Edge and Kernel Protection (Enterprise)

For enterprise-grade clients, BidShard supports high-capacity perimeter filtering before traffic even touches the application layer:

1. **Nginx / OpenResty Edge (`access-check.lua`):**
   Synchronizes manual, automatic, and fraud-related IP blacklists from Redis Shard 0 directly into Nginx shared memory. Blocks or rate-limits blacklisted IPs at the web-server gateway.
2. **eBPF/XDP Kernel-Level Drops:**
   Bypasses the entire Linux userspace network stack. Blacklisted IPs are dropped directly on the Network Interface Card (NIC) via XDP (eXpress Data Path), handling up to **hundreds of thousands of requests per second** with negligible CPU usage. Synchronizes increments asynchronously without locking or scanning full sets.

---

## Asynchronous Cold-Path Analysis

### IVT Detector (`cmd/ivt-detector`)
Scans ClickHouse analytics tables (`ml_features_1m`) every 5 minutes using specialized rules to detect macro-anomalies:
- **CTR Spikes:** Flags IPs exhibiting unrealistic Click-Through-Rates (CTR).
- **Cluster Detection:** Groups visitors sharing identical, highly unique fingerprint hashes.
- **Interval Bot Detection:** Exposes automated scripts clicking on fixed, perfect intervals.
- **Outbox Backpressure Guard:** Automatically pauses the IVT detector if the outbox queue grows too large, protecting the database from transaction storms.

### Fraud Scorer (`cmd/fraud-scorer`)
Uses pre-trained offline LightGBM or ONNX models to process complex multi-variable behavioral patterns in batches of up to 1,000 events. The resulting risk boosts are synced to Redis, ensuring the tracker always acts on fresh, machine-learning-driven data without sacrificing redirect speed.
