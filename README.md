# BidShard

A high-performance, self-hosted tracker, ad event processor, and real-time bidding (RTB) engine designed for media buyers, affiliate marketing teams, and programmatic ad networks.

`BidShard` is an enterprise-grade, independent on-premises platform built for campaign optimization, programmatic traffic ingestion, and intelligent fraud prevention. Unlike traditional cloud-based SaaS trackers, BidShard is deployed directly on your own infrastructure. This guarantees complete data privacy, protects your profitable campaigns and funnels from spying, and removes any arbitrary limits on campaigns or event volumes.

---

## Why Media Buyers Choose BidShard

### 1. Unlimited Campaigns, Clicks, and Offers
Stop paying extra for scaling your traffic. Traditional SaaS trackers penalize your success by bumping you into more expensive subscription tiers as your click volume grows. BidShard runs on your own hardware: you pay a flat, predictable license fee based on your peak Request-Per-Second (RPS) capability and host count. The number of campaigns, landing pages, offers, and overall monthly click volume is always completely unlimited.

### 2. High-Performance Architecture
Every millisecond of redirect latency eats into your Conversion Rate (CR) and drops traffic on transit. BidShard is engineered on an ultra-fast Go and Redis stack, meticulously optimized for memory safety with **zero heap allocations on the hot path**.

#### Low-Latency Engineering Details:
- **Fast Path Isolation:** Ingestion (`/track` and `/click`) is isolated from cold analytical storage writes. Events are accepted instantly, debited in Redis, and pushed to lock-free asynchronous rings.
- **Latency Benchmarks:** Under production load, average request processing time (**p95**) is **under 50 milliseconds**, and peak tail latency (**p99**) remains **under 80 milliseconds** (with a hard system ceiling of 100ms).
- **Asynchronous Logging:** The tracker uses a high-performance `StreamProducer` to buffer events asynchronously into Redis Streams using batch `XADD` pipelines, avoiding any blocking disk I/O or Postgres writes during the request lifecycle.
- **Go Pre-Filtering:** Cheap deterministic filters (License, Geo, Schedule, Segment, VPP) execute locally in-memory before evaluating heavier filters, minimizing Redis round-trips.
- **Local Quanta Full-Skip:** High-volume campaigns can activate in-memory quota allocations, cutting out synchronous Redis network hops entirely for ultra-low latency.

### 3. Margin Guard (Automatic ROI & Payout Protection)
An integrated safety net that shields your budget from technical mishaps. If an offer goes down on the affiliate network side, a landing page host crashes, or a campaign suffers a sudden spike in clicks without conversions, Margin Guard reacts instantly:
- Pauses the affected campaign or redirects traffic to your pre-configured safe fallback links.
- Locks down budget bleeding when the ROI drops below your custom threshold.
- Safely protects your working capital during overnight runs or unmonitored spikes.

### 4. Native Cost Sync and Conversions API (CAPI)
- **Cost Sync:** Automatically pull and synchronize ad spend from popular sources (Facebook, Google, TikTok, and more) directly into your dashboard for real-time, accurate net profit calculation.
- **CAPI Integration:** Seamlessly send conversion events back to ad networks (including Meta CAPI) directly from your server. This bypasses ad-blockers, iOS tracking restrictions, and browser third-party cookie limitations, ensuring your ad algorithms receive clean, complete attribution data.

---

## Advanced Antifraud & Landing Protection (GMA)

BidShard features a built-in, multi-layered filtering engine designed to block bots, crawlers, compliance moderators, and spy tools. The system proactively checks dozens of hot-path signals to keep your funnels clean and secure.

```
Incoming Request
  │
  ├── Perimeter (Nginx Lua / eBPF) -> IP Blacklist / Syn Fingerprints (Microseconds)
  │
  ├── Go FilterEngine (Tracker)    -> Local Filters: Geo, Schedule, Device (Nanoseconds)
  │                                -> Network/TLS: JA3/JA4 Fingerprints, TCP MSS Anomalies
  │
  └── Settlement & ML (Async)      -> ClickHouse Logging -> ivt-detector & fraud-scorer (ML Boost)
```

### Hot-Path Anti-Bot Signals & Network Auditing

BidShard evaluates incoming traffic against a comprehensive array of real-time signals:
1. **JA3/JA4 TLS Fingerprinting:** Identifies scraper bots and headless browsers that spoof their User-Agent but retain default curl, python, or Node.js TLS handshakes. Matches fingerprints against a fast local allowlist/blocklist.
2. **TCP MSS & Initial TTL Anomalies:** Analyzes lower-level packet headers (via `X-TCP-MSS` and `X-TCP-TTL` passed from the edge) using passive OS fingerprinting (`p0f-lite` rules). This exposes virtualization anomalies—such as a Windows User-Agent arriving with Linux network stack signatures.
3. **Sticky IP /24 & IPv6 /64 Rotation Limits:** Detects automated IP-rotation farms by monitoring click/impression velocity from matching subnets. Excessive rotation triggers L1 or L2 actions.
4. **Local Residential Proxy Farm Detection:** Maintains a hot-ring cache of campaign-local click signatures to immediately identify and segment proxy-relayed bot traffic.
5. **Datacenter & VPN IP Blocking:** Integrates a fast MaxMind GeoIP and custom ASN blacklist to intercept server-side cloud traffic (AWS, GCP, DigitalOcean, Tor) on the fly.

### Multi-Level Reaction System

* **L1 - Reject:** Hard-blocks known bots, bad subnets, and blacklisted IPs. `/click` requests route directly to safe-view landing pages or return a HTTP 204.
* **L2 - Shadow:** Admits suspicious traffic but tags it as a `shadow_event`. It is visible in reports and ClickHouse analytics so you can study bot behavior without risking false positives on real users.
* **Ghost IVT (Phantom Bypass):** An advanced protection mode. The tracker fake-accepts bot and spy tool requests, returning a successful status (HTTP 202/204) to mimic normal behavior. However, campaign budget is not debited, and conversion events or webhooks are never fired to ad networks. Bots think they successfully crawled your page, keeping your active funnels safely hidden.
* **L3 - Blacklist:** Instantly quarantines malicious IPs into the Redis fast lane, syncing blocking rules across all instances within milliseconds.

### Pre-configured Antifraud Presets

* **Social In-App (Optimized for Facebook, TikTok, Instagram WebViews):**
  A dedicated mode for social traffic. In-app mobile WebViews often have highly specific network fingerprints that can trigger false-positive bot alarms on standard trackers. This preset is fine-tuned for mobile carrier traffic, minimizing false blocks and keeping conversion rates as high as possible.
* **Gray Market (Max GMA Protection with JS Attestation):**
  The ultimate security mode. It deploys distributed Safe Page routing (cloaking) combined with interactive, lightweight JS browser attestation. On first click, the browser is silently audited for genuine human signatures (WebGL/canvas rendering, WebRTC audio-context capabilities, actual Bezier-curve mouse movements, or touch events), filtering out even the most sophisticated automated bot networks.

---

## Licensing & Tiers (USDT / Month)

Our licensing model is straightforward and transparent: you pay only for server node activations and peak RPS throughput. There are no monthly click quotas or SaaS caps.

| Tier | Price | Hosts (Nodes) | Peak RPS | Key Features |
| :--- | :--- | :--- | :--- | :--- |
| **Starter** | $129 | 1 | 10k | Core Tracking, Cost Sync, Meta CAPI, Margin Guard |
| **Pro** | $329 | 1 | 25k | All Starter features + OpenRTB engine for programmatic bidding |
| **Scale** | $649 | 3 | 75k | All Pro features + Machine Learning (AI Antifraud), live residential proxy intelligence |
| **Network** | $1,199 | 10 | 150k | All Scale features + Enterprise multi-region routing & failover |
| **Enterprise** | $2,500+ | 99 | Custom | All Network features + eBPF/XDP network-level filtering |
| **Pilot** | $0 | 1 | 5k | 10-day free access to test integrations and verify performance |

---

## Deployment Profiles

Choose the best deployment layout based on your available server hardware:

1. **ingest-only (Lightweight Stack):**
   - Runs exclusively on PostgreSQL and Redis. **No ClickHouse required.**
   - Tiny server footprint: works perfectly on servers with just **6-8 GB RAM**.
   - Ideal for solo buyers and agile teams who need lightning-fast redirects, Cost Sync, Margin Guard rules, and Meta CAPI without requiring heavy, raw event analytics tables. Reduces your monthly VPS hosting bills.

2. **single-vps (Full Analytical Stack):**
   - Deploys ClickHouse alongside the core components to collect raw traffic event logs.
   - Enables rich, multi-dimensional reporting (by placements, geos, devices, bot categories, and custom sub-IDs) in real time.
   - Recommended hardware: **16 GB RAM** or more.

---

## Quick Start

To request a 10-day **Pilot** license, reach out to your account representative on Telegram. Once approved, you will receive a `license.jwt` file. Simply drop this license file into your server's configured path.

For system administrators and DevOps engineers, detailed technical instructions, environment configurations, and setup guides are available in `docs/DEVELOPMENT.md`.

To explore the low-latency system architecture and hot/cold path engineering details, read `docs/ARCHITECTURE.md`.
